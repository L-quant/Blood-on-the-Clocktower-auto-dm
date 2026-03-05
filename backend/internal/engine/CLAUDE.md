# engine

## 职责
游戏状态机核心：命令分发 (28 种命令)、事件生成 (30+ 种事件)、状态归约、胜负判定

## 成员文件
- `engine.go` → 命令处理器总入口，路由所有命令到具体 handler (advance_phase 支持 DM 兜底权限)；handleAbility 仅记录意图，全部完成后触发三层流水线
- `engine_start_helpers.go` → handleStartGame 辅助函数：parseCustomRoles (payload 解析)、buildNoActionCompletions (首夜 no_action 自动完成)
- `engine_night_resolve.go` → 夜晚统一结算层：resolveNight (投毒→僧侣→恶魔击杀→红唇继承→投毒者死亡回滚)、applyResolveEffects (效果应用到 state 副本)
- `engine_night_info.go` → 夜晚信息分发层：distributeNightInfo (生成 night.info 事件)、generateTeamRecognition (首夜邪恶互认)、generateSpyGrimoire (间谍魔典)
- `engine_night_seq.go` → 夜晚行动排序：buildFirstPrompt / buildNextPrompt / validateCurrentNightAction
- `state.go` → 游戏状态结构体定义 (Player.SpyApparentRole, State.ScarletWomanTriggered, State.AwaitingRavenkeeper)、胜负检查、OwnerID 迁移
- `state_reduce.go` → Reduce 事件归约：处理 35+ 种事件 (含 night.info / team.recognition / poison.rollback)
- `vote_resolve.go` → 统一投票结算入口 (resolveVoteAndCheckWin)，含每日一次处决守卫 (ExecutedToday)，handleVote/handleCloseVote 共用
- `engine_extend.go` → extend_time 命令：白天讨论延长时间 (最多 MaxExtensions 次)
- `engine_night_timeout.go` → night_timeout 命令：差异化夜晚超时 (善良方自动完成，邪恶方发 action.reminder)
- `night_timeout.go` → 夜晚超时自动补全：按 ActionType 区分，info/good 自动 timed_out，evil critical (imp/poisoner) 跳过
- `engine_test.go` → 命令处理、游戏流程、action_type 验证测试
- `engine_extend_test.go` → extend_time 命令测试 (正常/超限/错误阶段/Reduce)
- `engine_night_timeout_test.go` → night_timeout 命令测试 (全完成→天亮/邪恶待定→提醒/错误阶段)
- `night_timeout_test.go` → 夜晚超时补全与 isEvilCriticalAction 测试
- `vote_resolve_test.go` → 投票结算、事件一致性、autodm 权限、阈值、OwnerID 迁移、DM 权限、每日一次处决测试
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
- `CompleteRemainingNightActions(state State, cmd types.CommandEnvelope) ([]types.Event, bool)` → 按 ActionType 补全未完成夜晚行动，返回 (事件, 是否有邪恶关键行动未完成)

## 依赖
- `internal/game` → 角色定义、夜晚行动解析 (NightAgent)、游戏初始化
- `internal/types` → CommandEnvelope、Event 类型
