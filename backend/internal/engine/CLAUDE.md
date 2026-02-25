# engine

## 职责
游戏状态机核心：命令分发 (28 种命令)、事件生成 (30+ 种事件)、状态归约、胜负判定

## 成员文件
- `engine.go` → 命令处理器总入口，路由所有命令到具体 handler
- `state.go` → 游戏状态结构体定义、Reduce 事件归约、胜负检查
- `engine_test.go` → 命令处理与基本游戏流程测试
- `scarlet_woman_test.go` → 恶魔继承 (Starpass) 与 Scarlet Woman 优先级测试
- `win_check_test.go` → 胜负条件测试 (恶魔死亡、人数不足、Saint、Mayor 等)

## 对外接口
- `HandleCommand(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error)` → 处理命令并返回事件列表
- `NewState(roomID string) State` → 创建初始游戏状态
- `DefaultGameConfig() GameConfig` → 返回默认阶段时长配置
- `(State) Copy() State` → 深拷贝游戏状态
- `(*State) Reduce(event EventPayload)` → 将事件应用到状态
- `(*State) GetAliveCount() int` → 统计存活非 DM 玩家数
- `(*State) GetAliveNeighbors(userID string) (left, right string)` → 获取相邻存活玩家
- `(*State) CheckWinCondition() (ended bool, winner, reason string)` → 检查游戏结束条件
- `MarshalState(s State) (string, error)` → 序列化状态为 JSON
- `UnmarshalState(raw string) (State, error)` → 从 JSON 反序列化状态

## 依赖
- `internal/game` → 角色定义、夜晚行动解析 (NightAgent)、游戏初始化
- `internal/types` → CommandEnvelope、Event 类型
