package main

import (
	"context"
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

	// Initialize AutoDM (AI Storyteller)
	slogLogger := observability.ZapToSlog(logger)
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
		Logger: slogLogger,
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
