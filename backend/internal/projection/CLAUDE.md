# projection

## 职责
事件可见性过滤与状态投影，按玩家角色过滤敏感信息 (如当前角色只能看到自己发动技能而看不到其他角色发送技能、无法看见其他玩家角色身份)

## 成员文件
- `projection.go` → 事件过滤 (Project) 与状态脱敏 (ProjectedState)；支持 night.info（仅目标玩家可见、strip is_false）、team.recognition（仅目标邪恶玩家可见、minion strip bluffs）、poison.rollback（不可见）

## 对外接口
- `Project(event types.Event, state engine.State, viewer types.Viewer) *types.ProjectedEvent` → 按观察者过滤单个事件，返回 nil 表示不可见
- `ProjectedState(state engine.State, viewer types.Viewer) engine.State` → 返回脱敏后的游戏状态副本

## 依赖
- `internal/engine` → State 结构体用于状态脱敏
- `internal/types` → Event、Viewer、ProjectedEvent 类型
