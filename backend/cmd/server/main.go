package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/api"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/auth"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/bot"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/config"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/observability"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/queue"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/rag"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/realtime"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/room"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/store"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"

	_ "github.com/qingchang/Blood-on-the-Clocktower-auto-dm/docs" // Import swagger docs
)

func main() {
	cfg := config.Load()
	logger, err := observability.SetupLogger()
	if err != nil {
		log.Fatalf("cannot init logger: %v", err)
	}
	defer logger.Sync()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tp, err := observability.SetupTracerProvider(ctx, "agentdm", cfg.TraceStdout, logger)
	if err != nil {
		logger.Fatal("cannot init tracer", zap.Error(err))
	}
	defer tp.Shutdown(ctx)

	db, err := store.ConnectMySQL(cfg.DBDSN)
	if err != nil {
		logger.Fatal("cannot connect db", zap.Error(err))
	}
	defer db.Close()
	st := store.New(db)

	metrics := observability.NewMetrics(prometheus.DefaultRegisterer.(*prometheus.Registry))
	jwtMgr := auth.NewJWTManager(cfg.JWTSecret, 24*time.Hour)

	// Initialize RAG system
	var retriever *rag.RuleRetriever
	if cfg.QdrantHost != "" {
		qdrantClient := rag.NewQdrantClient(cfg.QdrantHost, cfg.QdrantPort, cfg.QdrantCollection)

		var embedder rag.EmbeddingProvider
		if cfg.AutoDMLLMProvider == "gemini" {
			embedder = rag.NewGeminiEmbedding(rag.GeminiEmbeddingConfig{
				APIKey:     cfg.GeminiAPIKey,
				BaseURL:    cfg.AutoDMLLMBaseURL,
				Dimensions: 768,
			})
		} else {
			embedder = rag.NewOpenAIEmbedding(rag.OpenAIEmbeddingConfig{
				APIKey:     cfg.AutoDMLLMAPIKey,
				BaseURL:    cfg.AutoDMLLMBaseURL,
				Dimensions: 1536,
			})
		}
		retriever = rag.NewRuleRetriever(qdrantClient, embedder)

		// Initialize with rules from docs/rules directory
		rulesDir := "../docs/rules"
		if err := retriever.Initialize(ctx, rulesDir); err != nil {
			logger.Warn("Failed to initialize RAG", zap.Error(err))
		} else {
			logger.Info("RAG system initialized", zap.String("rules_dir", rulesDir))
		}
	}

	// Initialize task queue
	var taskQueue *queue.Queue
	if cfg.RabbitMQURL != "" {
		slogLogger := observability.ZapToSlog(logger)
		taskQueue, err = queue.New(queue.Config{
			URL:       cfg.RabbitMQURL,
			QueueName: "agentdm_tasks",
			Prefetch:  10,
			Logger:    slogLogger,
		})
		if err != nil {
			logger.Warn("Failed to connect to RabbitMQ", zap.Error(err))
		} else {
			logger.Info("Task queue connected", zap.String("url", cfg.RabbitMQURL))
			defer taskQueue.Close()
		}
	}

	// Initialize AutoDM (AI Storyteller)
	slogLogger := observability.ZapToSlog(logger)

	// Create adapters for interfaces
	var retrieverAdapter agent.RuleRetriever
	if retriever != nil {
		retrieverAdapter = &ruleRetrieverAdapter{r: retriever}
	}
	var taskQueueAdapter agent.TaskQueue
	if taskQueue != nil {
		taskQueueAdapter = &taskQueueAdapterImpl{q: taskQueue}
	}

	autoDM := agent.NewAutoDM(agent.Config{
		RoomID:  "", // Will be set per-room
		Enabled: cfg.AutoDMEnabled,
		LLM: agent.LLMRoutingConfig{
			Default: agent.LLMClientConfig{
				BaseURL:    cfg.AutoDMLLMBaseURL,
				APIKey:     cfg.AutoDMLLMAPIKey,
				Model:      cfg.AutoDMLLMModel,
				Timeout:    cfg.AutoDMLLMTimeout,
				HTTPSProxy: cfg.HTTPSProxy,
			},
		},
		Logger:    slogLogger,
		Retriever: retrieverAdapter,
		TaskQueue: taskQueueAdapter,
	})

	if autoDM.Enabled() {
		logger.Info("AutoDM enabled",
			zap.String("provider", cfg.AutoDMLLMProvider),
			zap.String("model", cfg.AutoDMLLMModel),
			zap.String("base_url", cfg.AutoDMLLMBaseURL))
	}

	roomMgr := room.NewRoomManager(ctx, st, logger, metrics, cfg.SnapshotInterval, autoDM)
	defer roomMgr.Close()
	if autoDM.Enabled() {
		autoDM.SetDispatcher(roomMgr, nil)
		autoDM.Start()
		defer autoDM.Stop()
	}

	if taskQueue != nil {
		taskQueue.RegisterHandler("autodm_event", func(ctx context.Context, task queue.Task) (map[string]interface{}, error) {
			raw, ok := task.Data["event"]
			if !ok {
				return nil, fmt.Errorf("task data missing event field")
			}

			var eventJSON []byte
			switch v := raw.(type) {
			case string:
				eventJSON = []byte(v)
			default:
				b, err := json.Marshal(v)
				if err != nil {
					return nil, err
				}
				eventJSON = b
			}

			var ev types.Event
			if err := json.Unmarshal(eventJSON, &ev); err != nil {
				return nil, err
			}
			if err := autoDM.ProcessQueuedEvent(ctx, ev); err != nil {
				return nil, err
			}

			return map[string]interface{}{
				"status": "processed",
				"room":   ev.RoomID,
				"type":   ev.EventType,
			}, nil
		})

		if err := taskQueue.Start(ctx); err != nil {
			logger.Error("Failed to start task queue", zap.Error(err))
		}
	}

	botMgr := bot.NewManager(observability.ZapToSlog(logger))

	wsServer := realtime.NewWSServer(jwtMgr, st, roomMgr, logger, metrics)
	server := api.NewServer(st, jwtMgr, roomMgr, wsServer, logger,
		api.WithLLMInfo(&api.LLMInfo{
			Provider: cfg.AutoDMLLMProvider,
			Model:    cfg.AutoDMLLMModel,
			BaseURL:  cfg.AutoDMLLMBaseURL,
			Enabled:  cfg.AutoDMEnabled,
		}),
		api.WithBotManager(botMgr),
	)

	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: server.Router}
	go func() {
		logger.Info("starting server", zap.String("addr", cfg.HTTPAddr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)
}

// ruleRetrieverAdapter adapts rag.RuleRetriever to agent.RuleRetriever
type ruleRetrieverAdapter struct {
	r *rag.RuleRetriever
}

func (a *ruleRetrieverAdapter) Retrieve(ctx context.Context, query string, limit int) ([]agent.RetrieveResult, error) {
	results, err := a.r.Retrieve(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	converted := make([]agent.RetrieveResult, len(results))
	for i, r := range results {
		converted[i] = agent.RetrieveResult{
			Content:  r.Content,
			Score:    r.Score,
			Metadata: r.Metadata,
		}
	}
	return converted, nil
}

// taskQueueAdapterImpl adapts queue.Queue to agent.TaskQueue
type taskQueueAdapterImpl struct {
	q *queue.Queue
}

func (a *taskQueueAdapterImpl) Publish(ctx context.Context, task interface{}) error {
	switch t := task.(type) {
	case queue.Task:
		return a.q.Publish(ctx, t)
	case agent.AsyncEventTask:
		eventJSON, err := json.Marshal(t.Event)
		if err != nil {
			return err
		}
		qt := queue.Task{
			ID:        uuid.NewString(),
			Type:      t.Type,
			RoomID:    t.RoomID,
			Data:      map[string]interface{}{"event": string(eventJSON)},
			Priority:  7,
			CreatedAt: time.Now().UTC(),
			MaxRetry:  3,
		}
		return a.q.Publish(ctx, qt)
	default:
		return fmt.Errorf("invalid task type")
	}
}
