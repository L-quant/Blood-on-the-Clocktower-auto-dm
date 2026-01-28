package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/api"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/auth"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/config"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/observability"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/queue"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/rag"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/realtime"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/room"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/store"
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
		embedder := rag.NewOpenAIEmbedding(rag.OpenAIEmbeddingConfig{
			APIKey:     cfg.AutoDMLLMAPIKey,
			BaseURL:    cfg.AutoDMLLMBaseURL,
			Dimensions: 1536,
		})
		retriever = rag.NewRuleRetriever(qdrantClient, embedder)

		// Initialize with rules from assets/rules directory
		rulesDir := "../assets/rules"
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

			// Start consuming tasks
			if err := taskQueue.Start(ctx); err != nil {
				logger.Error("Failed to start task queue", zap.Error(err))
			}
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
				BaseURL: cfg.AutoDMLLMBaseURL,
				APIKey:  cfg.AutoDMLLMAPIKey,
				Model:   cfg.AutoDMLLMModel,
				Timeout: cfg.AutoDMLLMTimeout,
			},
		},
		Logger:    slogLogger,
		Retriever: retrieverAdapter,
		TaskQueue: taskQueueAdapter,
	})

	if autoDM.Enabled() {
		logger.Info("AutoDM enabled",
			zap.String("model", cfg.AutoDMLLMModel),
			zap.String("base_url", cfg.AutoDMLLMBaseURL))
	}

	roomMgr := room.NewRoomManager(st, logger, metrics, cfg.SnapshotInterval, autoDM)
	wsServer := realtime.NewWSServer(jwtMgr, st, roomMgr, logger, metrics)
	server := api.NewServer(st, jwtMgr, roomMgr, wsServer, logger)

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
	t, ok := task.(queue.Task)
	if !ok {
		return fmt.Errorf("invalid task type")
	}
	return a.q.Publish(ctx, t)
}
