# bot

## 职责
AI Bot 玩家实现，支持基于性格的自动决策 (发言、投票、提名) 与生命周期管理

## 成员文件
- `bot.go` → 单个 Bot 玩家逻辑，性格驱动的决策 (aggressive/cautious/random/smart)
- `manager.go` → Bot 生命周期管理，跨房间创建/分发事件/移除
- `bot_test.go` → Bot 与 Manager 的单元测试

## 对外接口
- `NewBot(cfg BotConfig) *Bot` → 创建 Bot 玩家
- `(*Bot) UserID() string` → 返回 Bot 用户 ID
- `(*Bot) Name() string` → 返回 Bot 显示名
- `(*Bot) SetDispatcher(d CommandDispatcher, roomID string)` → 设置命令分发器
- `(*Bot) OnEvent(ctx context.Context, ev types.Event)` → 处理游戏事件并自动响应
- `NewManager(logger *slog.Logger) *Manager` → 创建 Bot 管理器
- `(*Manager) AddBots(ctx context.Context, req AddBotsRequest, dispatcher CommandDispatcher) ([]string, error)` → 向房间添加 Bot (最多 14 个)
- `(*Manager) OnEvent(ctx context.Context, roomID string, ev types.Event)` → 向房间所有 Bot 广播事件
- `(*Manager) GetBots(roomID string) []*Bot` → 获取房间内所有 Bot
- `(*Manager) RemoveBots(roomID string)` → 移除房间所有 Bot
- `(*Manager) BotCount(roomID string) int` → 返回房间 Bot 数量

## 依赖
- `internal/types` → CommandEnvelope、Event 类型
