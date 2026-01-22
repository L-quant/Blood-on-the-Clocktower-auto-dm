package observability

import (
	"context"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.uber.org/zap"
)

type Metrics struct {
	ActiveConnections prometheus.Gauge
	RoomQueueLen      *prometheus.GaugeVec
	CommandLatency    *prometheus.HistogramVec
	DBTxLatency       prometheus.Observer
	BroadcastLatency  prometheus.Observer
	DedupHitTotal     prometheus.Counter
	CommandReject     *prometheus.CounterVec
	ResyncEvents      prometheus.Counter
	AgentLatency      prometheus.Observer
	AgentErrorTotal   prometheus.Counter
}

func NewMetrics(reg *prometheus.Registry) *Metrics {
	if reg == nil {
		reg = prometheus.DefaultRegisterer.(*prometheus.Registry)
	}
	return &Metrics{
		ActiveConnections: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "ws_active_connections",
			Help: "Number of active websocket connections",
		}),
		RoomQueueLen: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Name: "room_actor_queue_len",
			Help: "Buffered commands waiting per room actor",
		}, []string{"room_id"}),
		CommandLatency: promauto.With(reg).NewHistogramVec(prometheus.HistogramOpts{
			Name:    "command_latency_ms",
			Help:    "Latency for processing commands",
			Buckets: prometheus.ExponentialBuckets(1, 2, 12),
		}, []string{"command_type"}),
		DBTxLatency: promauto.With(reg).NewHistogram(prometheus.HistogramOpts{
			Name:    "db_tx_latency_ms",
			Help:    "DB transaction latency",
			Buckets: prometheus.ExponentialBuckets(1, 2, 12),
		}),
		BroadcastLatency: promauto.With(reg).NewHistogram(prometheus.HistogramOpts{
			Name:    "broadcast_latency_ms",
			Help:    "Broadcast latency",
			Buckets: prometheus.ExponentialBuckets(1, 2, 12),
		}),
		DedupHitTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "dedup_hit_total",
			Help: "Number of dedup hits",
		}),
		CommandReject: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "command_reject_total",
			Help: "Rejected commands",
		}, []string{"reason"}),
		ResyncEvents: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "resync_events_total",
			Help: "Events resynced to clients",
		}),
		AgentLatency: promauto.With(reg).NewHistogram(prometheus.HistogramOpts{
			Name:    "agent_run_latency_ms",
			Help:    "Agent run latency",
			Buckets: prometheus.ExponentialBuckets(1, 2, 12),
		}),
		AgentErrorTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "agent_error_total",
			Help: "Agent run errors",
		}),
	}
}

func SetupTracerProvider(ctx context.Context, serviceName string, stdout bool, logger *zap.Logger) (*sdktrace.TracerProvider, error) {
	var exporter *stdouttrace.Exporter
	var err error
	if stdout {
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, err
		}
	}

	rs := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(rs),
	)
	if exporter != nil {
		tp.RegisterSpanProcessor(sdktrace.NewBatchSpanProcessor(exporter))
	}
	otel.SetTracerProvider(tp)
	logger.Info("tracer initialized")
	return tp, nil
}

func SetupLogger() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Encoding = "json"
	return cfg.Build()
}

// ZapToSlog wraps a zap.Logger as slog.Logger.
func ZapToSlog(logger *zap.Logger) *slog.Logger {
	return slog.New(slogHandler{logger.Sugar()})
}

type slogHandler struct {
	sugar *zap.SugaredLogger
}

func (h slogHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h slogHandler) Handle(ctx context.Context, r slog.Record) error {
	args := make([]interface{}, 0, r.NumAttrs()*2)
	r.Attrs(func(a slog.Attr) bool {
		args = append(args, a.Key, a.Value.Any())
		return true
	})
	switch r.Level {
	case slog.LevelDebug:
		h.sugar.Debugw(r.Message, args...)
	case slog.LevelInfo:
		h.sugar.Infow(r.Message, args...)
	case slog.LevelWarn:
		h.sugar.Warnw(r.Message, args...)
	case slog.LevelError:
		h.sugar.Errorw(r.Message, args...)
	}
	return nil
}

func (h slogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	args := make([]interface{}, 0, len(attrs)*2)
	for _, a := range attrs {
		args = append(args, a.Key, a.Value.Any())
	}
	return slogHandler{h.sugar.With(args...)}
}

func (h slogHandler) WithGroup(name string) slog.Handler {
	return h
}
