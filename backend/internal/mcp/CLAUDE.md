# mcp

## 职责
Model Context Protocol 工具注册表，管理 AI 代理可调用的工具定义、执行与审计日志

## 成员文件
- `registry.go` → 工具注册、查询、同步/异步执行、任务管理、审计日志
- `tools.go` → 游戏工具定义与注册 (发消息、推进阶段、提名等 11 个工具)

## 对外接口
- `NewRegistry() *Registry` → 创建工具注册表
- `(*Registry) Register(def ToolDefinition, handler ToolHandler) error` → 注册工具
- `(*Registry) GetTool(name string) (ToolDefinition, bool)` → 按名称查询工具
- `(*Registry) ListTools() []ToolDefinition` → 列出所有工具
- `(*Registry) ListToolsByCategory(category ToolCategory) []ToolDefinition` → 按类别过滤工具
- `(*Registry) Invoke(ctx context.Context, call ToolCall) *ToolResult` → 执行工具
- `(*Registry) GetTask(taskID string) (*AsyncTask, bool)` → 查询异步任务
- `(*Registry) TaskChannel() <-chan *AsyncTask` → 获取任务完成通知通道
- `NewAuditor() *Auditor` → 创建审计日志记录器
- `(*Auditor) Record(roomID, agentID string, call ToolCall, result *ToolResult, duration time.Duration)` → 记录工具调用日志
- `(*Auditor) GetLogs(roomID string, limit int) []AuditLog` → 查询审计日志
- `RegisterGameTools(registry *Registry, cfg GameToolsConfig) error` → 注册所有游戏工具

## 依赖
- `internal/types` → CommandEnvelope 类型 (工具执行时构建命令)
