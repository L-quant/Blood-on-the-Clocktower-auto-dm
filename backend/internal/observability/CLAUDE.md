# observability

## 职责
可观测性基础设施：Prometheus 指标采集、OpenTelemetry 分布式追踪、Zap 日志初始化

## 成员文件
- `observability.go` → Metrics 初始化 (10 个指标)、TracerProvider 配置、Logger 创建、Zap→Slog 适配

## 对外接口
- `NewMetrics(reg *prometheus.Registry) *Metrics` → 初始化 Prometheus 指标 (WS 连接数、命令延迟、DB 事务延迟、广播延迟等)
- `SetupTracerProvider(ctx context.Context, serviceName string, stdout bool, logger *zap.Logger) (*sdktrace.TracerProvider, error)` → 初始化 OTel 追踪
- `SetupLogger() (*zap.Logger, error)` → 配置生产级 Zap 日志器
- `ZapToSlog(logger *zap.Logger) *slog.Logger` → 将 Zap 包装为 slog 适配器

## 依赖
无内部依赖
