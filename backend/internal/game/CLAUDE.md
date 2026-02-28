# game

## 职责
角色定义、夜晚能力解析、游戏初始化 (分配角色/夜晚顺序)，自包含无内部依赖

## 成员文件
- `roles.go` → 定义所有暗流涌动角色 (含 ActionType: info/select_one/select_two/no_action)、玩家分配表
- `night.go` → 夜晚能力解析引擎，处理 13 种能力 (含中毒/保护逻辑)
- `setup.go` → 游戏初始化：角色分配 (支持 CustomRoles 和随机选择)、Baron 自动检测 (+2 outsider)、夜晚顺序创建
- `compose.go` → 角色组合接口 (Composer)、RandomComposer (随机选角)、FallbackComposer (主→备降级)
- `night_test.go` → 夜晚能力解析的 24 个测试用例
- `setup_test.go` → Baron 修正、CustomRoles、RandomComposer、FallbackComposer 测试 (7 tests)

## 对外接口
- `GetRoleByID(id string) *Role` → 按 ID 查询角色
- `GetRolesByType(roleType RoleType) []Role` → 按类型获取角色列表
- `GetAllRoles() []Role` → 获取所有暗流涌动角色
- `GetDistribution(playerCount int) *PlayerDistribution` → 获取玩家数量对应的角色分配
- `GetNightOrder(firstNight bool) []Role` → 获取夜晚行动顺序
- `NewNightAgent(ctx *GameContext) *NightAgent` → 创建夜晚能力解析器
- `(*NightAgent) ResolveAbility(req AbilityRequest) (*AbilityResult, error)` → 解析角色夜晚能力
- `NewSetupAgent(config SetupConfig) *SetupAgent` → 创建游戏初始化代理
- `(*SetupAgent) GenerateAssignments(userIDs []string, seatOrder []int) (*SetupResult, error)` → 分配角色给玩家
- `GenerateNightOrder(roles []Role, assignments map[string]Assignment, firstNight bool) []NightAction` → 生成夜晚唤醒顺序
- `Composer` 接口 → `ComposeRoles(ctx, ComposeRequest) (*ComposeResult, error)` 角色组合
- `RandomComposer` → 基于标准分配表随机选角 (含 Baron 自动检测)
- `FallbackComposer` → 尝试主 Composer，失败回退到备用 Composer

## 依赖
无内部依赖
