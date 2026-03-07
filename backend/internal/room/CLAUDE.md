# room

## 职责
房间 Actor 模型：每房间独立命令队列串行处理，管理游戏状态、事件持久化、订阅者广播和自动快照

## 成员文件
- `room.go` → RoomActor (命令队列、状态管理、事件广播、重启计时器恢复) 与 RoomManager。计时器行为：白天讨论→提名 (非直接入夜)、nomination.resolved→NominationPhaseDurationSec、time.extended 重调度；夜晚超时路径当前版本显式禁用。start_game 命令拦截调用 Composer
- `room_config.go` → RoomDeps 配置结构体 (Store/Logger/Metrics/SnapshotInterval/AutoDM/Composer)，减少 NewRoomActor/NewRoomManager 参数数量
- `room_compose.go` → enrichStartGame：拦截 start_game 命令，调用 game.Composer 生成角色列表注入 custom_roles (15s 超时，失败回退随机)
- `phase_timer.go` → 阶段超时计时器 (PhaseTimer)，含 IdempotencyKey 和 generation 抗竞态保护
- `phase_timer_test.go` → PhaseTimer 单元测试 + 重启后计时器恢复测试
- `schedule_timeouts_test.go` → scheduleTimeouts 集成测试 (含 nomination.resolved 分支)

## 对外接口
- `NewRoomActor(loadCtx, loopCtx context.Context, roomID string, deps RoomDeps, onCrash func(string)) (*RoomActor, error)` → 创建房间 Actor 并加载持久化状态
- `(*RoomActor) Subscribe(id string, s *Subscriber)` → 注册 WebSocket 订阅者
- `(*RoomActor) Unsubscribe(id string)` → 移除订阅者
- `(*RoomActor) Dispatch(cmd types.CommandEnvelope) CommandResponse` → 同步分发命令并等待响应
- `(*RoomActor) DispatchAsync(cmd types.CommandEnvelope) error` → 异步分发命令 (不阻塞)
- `(*RoomActor) GetState() engine.State` → 获取当前游戏状态的线程安全副本
- `NewRoomManager(ctx context.Context, deps RoomDeps) *RoomManager` → 创建房间管理器
- `(*RoomManager) Close()` → 停止所有房间 Actor
- `(*RoomManager) GetOrCreate(ctx context.Context, roomID string) (*RoomActor, error)` → 获取或创建房间 Actor
- `(*RoomManager) DispatchAsync(cmd types.CommandEnvelope) error` → 按 RoomID 路由命令到对应 Actor
- `NewPhaseTimer(roomID string, dispatch func(types.CommandEnvelope), logger *zap.Logger) *PhaseTimer` → 创建阶段计时器
- `(*PhaseTimer) Schedule(dur time.Duration, cmdType string, data map[string]string)` → 调度超时命令 (自动取消上一个)
- `(*PhaseTimer) Cancel()` → 取消当前计时器

## 依赖
- `internal/agent` → AutoDM 集成 (事件回调)
- `internal/game` → Composer 角色组合接口
- `internal/engine` → HandleCommand 命令处理、State 状态归约
- `internal/observability` → 指标采集 (队列长度等)
- `internal/projection` → 事件广播前过滤
- `internal/store` → 事件持久化与快照
- `internal/types` → CommandEnvelope、Event 类型
