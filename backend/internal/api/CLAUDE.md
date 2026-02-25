# api

## 职责
HTTP REST API 路由与处理器，提供认证、房间管理、事件查询、状态同步和 WebSocket 入口

## 成员文件
- `api.go` → HTTP 服务器初始化、路由注册、所有 API 处理器实现

## 对外接口
- `NewServer(st *store.Store, jwt *auth.JWTManager, roomMgr *room.RoomManager, wsServer *realtime.WSServer, logger *zap.Logger, opts ...ServerOption) *Server` → 创建 HTTP 服务器并注册所有路由
- `WithLLMInfo(info *LLMInfo) ServerOption` → 配置 LLM 健康检查信息
- `WithBotManager(mgr *bot.Manager) ServerOption` → 配置 Bot 管理器

## 依赖
- `internal/auth` → JWT 令牌生成/验证、密码哈希
- `internal/bot` → Bot 玩家管理
- `internal/engine` → 游戏状态与事件 payload 结构
- `internal/projection` → 按角色过滤状态 (ProjectedState)
- `internal/realtime` → WebSocket 服务器集成
- `internal/room` → 房间管理器，获取房间状态
- `internal/store` → 用户/房间/事件数据库操作
- `internal/types` → Viewer 结构用于权限过滤
