# game

## 职责
角色定义、夜晚能力解析、游戏初始化 (分配角色/夜晚顺序)，自包含无内部依赖

## 成员文件
- `roles.go` → 定义所有暗流涌动 (Trouble Brewing) 角色、类型、能力、玩家分配表
- `night.go` → 夜晚能力解析引擎，处理 13 种能力 (含中毒/保护逻辑)
- `setup.go` → 游戏初始化：角色分配、夜晚顺序创建
- `night_test.go` → 夜晚能力解析的 24 个测试用例

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

## 依赖
无内部依赖
