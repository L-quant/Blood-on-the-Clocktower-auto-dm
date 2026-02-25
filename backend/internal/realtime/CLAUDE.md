# realtime

## 职责
WebSocket 服务器，管理客户端连接、房间订阅、事件推送 (含可见性过滤) 和命令转发，内置令牌桶限流

## 成员文件
- `ws.go` → WebSocket 升级、Session 管理、消息路由 (ping/subscribe/command)、令牌桶限流

## 对外接口
- `NewWSServer(jwt *auth.JWTManager, st *store.Store, roomMgr *room.RoomManager, logger *zap.Logger, metrics *observability.Metrics) *WSServer` → 创建 WebSocket 服务器
- `(*WSServer) ServeHTTP(w http.ResponseWriter, r *http.Request)` → HTTP 处理器，升级为 WebSocket 连接
- `NewTokenBucket(capacity, rate float64) *TokenBucket` → 创建令牌桶限流器
- `(*TokenBucket) Allow() bool` → 检查是否允许请求通过

## 依赖
- `internal/auth` → JWT 验证 WebSocket 连接
- `internal/observability` → 指标采集 (连接数等)
- `internal/projection` → 按观察者过滤事件
- `internal/room` → RoomManager 订阅房间事件
- `internal/store` → 加载历史事件
- `internal/types` → Viewer、ProjectedEvent 类型
