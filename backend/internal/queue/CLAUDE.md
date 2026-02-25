# queue

## 职责
RabbitMQ 异步任务队列，支持优先级、重试、死信队列，用于 AI 代理任务 (LLM 调用、RAG 查询、夜晚解析)

## 成员文件
- `queue.go` → 队列核心：连接管理、发布/消费、重试逻辑、死信队列
- `tasks.go` → 任务工厂与处理器：LLM 调用、RAG 查询、夜晚解析、TTS、摘要

## 对外接口
- `New(cfg Config) (*Queue, error)` → 创建并初始化 RabbitMQ 队列
- `(*Queue) RegisterHandler(taskType string, handler TaskHandler)` → 注册任务处理器
- `(*Queue) Publish(ctx context.Context, task Task) error` → 发布任务到队列
- `(*Queue) Start(ctx context.Context) error` → 开始消费任务
- `(*Queue) Results() <-chan TaskResult` → 获取任务结果通道
- `(*Queue) Close() error` → 关闭队列连接
- `(*Queue) HealthCheck() error` → 检查队列连接健康状态
- `NewTaskFactory() *TaskFactory` → 创建任务工厂
- `(*TaskFactory) CreateLLMCallTask(roomID string, data LLMCallData) Task` → 创建 LLM 调用任务
- `(*TaskFactory) CreateRAGQueryTask(roomID string, data RAGQueryData) Task` → 创建 RAG 查询任务
- `(*TaskFactory) CreateNightResolveTask(roomID string, data NightResolveData) Task` → 创建高优先级夜晚解析任务
- `(*TaskFactory) CreateSummarizeTask(roomID string, context map[string]interface{}) Task` → 创建低优先级摘要任务
- `(*AgentTaskHandlers) RegisterHandlers(q *Queue)` → 批量注册代理任务处理器

## 依赖
无内部依赖
