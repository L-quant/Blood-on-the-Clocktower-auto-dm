# rag

## 职责
规则文档的向量化检索 (RAG)：Embedding 生成 (OpenAI/Gemini/Local)、Qdrant 向量库交互、语义搜索

## 成员文件
- `embedding.go` → Embedding 生成器：OpenAI、Gemini、本地哈希 (测试用)
- `retriever.go` → 规则文档索引与语义检索，支持元数据过滤
- `client.go` → Qdrant 向量数据库 HTTP 客户端

## 对外接口
- `NewOpenAIEmbedding(cfg OpenAIEmbeddingConfig) *OpenAIEmbedding` → 创建 OpenAI Embedding 提供器
- `NewGeminiEmbedding(cfg GeminiEmbeddingConfig) *GeminiEmbedding` → 创建 Gemini Embedding 提供器
- `NewLocalEmbedding(dimensions int) *LocalEmbedding` → 创建本地测试用 Embedding
- `NewQdrantClient(host string, port int, collection string) *QdrantClient` → 创建 Qdrant 客户端
- `(*QdrantClient) EnsureCollection(ctx context.Context, vectorSize int) error` → 确保集合存在
- `(*QdrantClient) Upsert(ctx context.Context, points []Point) error` → 插入/更新向量点
- `(*QdrantClient) Search(ctx context.Context, vector []float64, limit int, filter map[string]interface{}) ([]SearchResult, error)` → 向量相似搜索
- `(*QdrantClient) Delete(ctx context.Context, ids []string) error` → 删除向量点
- `(*QdrantClient) Count(ctx context.Context) (int64, error)` → 统计向量点数量
- `NewRuleRetriever(qdrant *QdrantClient, embedder EmbeddingProvider) *RuleRetriever` → 创建规则检索器
- `(*RuleRetriever) Initialize(ctx context.Context, rulesDir string) error` → 初始化集合并索引规则文档
- `(*RuleRetriever) Retrieve(ctx context.Context, query string, limit int) ([]RetrieveResult, error)` → 语义检索规则
- `(*RuleRetriever) RetrieveWithFilter(ctx context.Context, query string, limit int, filter map[string]interface{}) ([]RetrieveResult, error)` → 带过滤条件的检索
- `(*RuleRetriever) IndexRoleRules(ctx context.Context, roleID, roleName, rules string) error` → 索引角色专属规则
- `(*RuleRetriever) GetRoleRules(ctx context.Context, roleID string) ([]RetrieveResult, error)` → 按角色 ID 检索规则

## 依赖
无内部依赖
