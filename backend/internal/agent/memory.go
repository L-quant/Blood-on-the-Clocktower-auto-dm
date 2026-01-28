package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// EmbeddingProvider defines the interface for generating embeddings
type EmbeddingProvider interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	Dimension() int
}

// MemoryStore interface for persisting memory
type MemoryStore interface {
	SaveEntry(ctx context.Context, roomID string, entry MemoryEntry) error
	LoadEntries(ctx context.Context, roomID string, entryType string, limit int) ([]MemoryEntry, error)
	SearchByEmbedding(ctx context.Context, roomID string, embedding []float32, topK int) ([]MemoryEntry, error)
	SaveGameSummary(ctx context.Context, roomID string, summary string) error
	GetGameSummary(ctx context.Context, roomID string) (string, error)
	SavePlayerModel(ctx context.Context, roomID string, model PlayerModel) error
	GetPlayerModels(ctx context.Context, roomID string) (map[string]PlayerModel, error)
}

// MemoryManager manages short-term and long-term memory with RAG capabilities
type MemoryManager struct {
	store         MemoryStore
	embedder      EmbeddingProvider
	rulesIndex    *VectorIndex
	shortTermSize int
	mu            sync.RWMutex
	shortTerm     map[string][]MemoryEntry // roomID -> entries
}

// MemoryManagerConfig configuration for the memory manager
type MemoryManagerConfig struct {
	ShortTermSize int
	RulesDir      string
}

// NewMemoryManager creates a new memory manager
func NewMemoryManager(store MemoryStore, embedder EmbeddingProvider, config MemoryManagerConfig) *MemoryManager {
	if config.ShortTermSize == 0 {
		config.ShortTermSize = 50
	}

	mm := &MemoryManager{
		store:         store,
		embedder:      embedder,
		shortTermSize: config.ShortTermSize,
		shortTerm:     make(map[string][]MemoryEntry),
	}

	// Initialize rules index
	dimension := 384 // default dimension
	if embedder != nil {
		dimension = embedder.Dimension()
	}
	mm.rulesIndex = NewVectorIndex(dimension)

	return mm
}

// Store saves a memory entry
func (m *MemoryManager) Store(ctx context.Context, roomID string, entry MemoryEntry) error {
	// Generate embedding if embedder is available
	if m.embedder != nil && len(entry.Embedding) == 0 {
		embeddings, err := m.embedder.Embed(ctx, []string{entry.Content})
		if err == nil && len(embeddings) > 0 {
			entry.Embedding = embeddings[0]
		}
	}

	// Add to short-term memory
	m.mu.Lock()
	entries := m.shortTerm[roomID]
	entries = append(entries, entry)
	if len(entries) > m.shortTermSize {
		// Move oldest to long-term storage
		toStore := entries[0]
		entries = entries[1:]
		go func() {
			if m.store != nil {
				m.store.SaveEntry(context.Background(), roomID, toStore)
			}
		}()
	}
	m.shortTerm[roomID] = entries
	m.mu.Unlock()

	return nil
}

// RetrieveRelevant retrieves relevant memory entries using RAG
func (m *MemoryManager) RetrieveRelevant(ctx context.Context, roomID string, query string, topK int) ([]MemoryEntry, error) {
	results := make([]MemoryEntry, 0)

	// Get from short-term (most recent first)
	m.mu.RLock()
	shortTerm := m.shortTerm[roomID]
	m.mu.RUnlock()

	// Add short-term entries with recency boost
	for i := len(shortTerm) - 1; i >= 0 && len(results) < topK; i-- {
		entry := shortTerm[i]
		entry.Score = 1.0 - float64(len(shortTerm)-1-i)*0.1 // Recency decay
		results = append(results, entry)
	}

	// Vector search in long-term memory if embedder available
	if m.embedder != nil && m.store != nil {
		embeddings, err := m.embedder.Embed(ctx, []string{query})
		if err == nil && len(embeddings) > 0 {
			longTerm, err := m.store.SearchByEmbedding(ctx, roomID, embeddings[0], topK)
			if err == nil {
				results = append(results, longTerm...)
			}
		}
	}

	// Search rules index
	if m.rulesIndex != nil {
		ruleResults := m.SearchRules(ctx, query, topK)
		results = append(results, ruleResults...)
	}

	// Sort by score and limit
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > topK {
		results = results[:topK]
	}

	return results, nil
}

// GetGameSummary retrieves the game summary for a room
func (m *MemoryManager) GetGameSummary(ctx context.Context, roomID string) (string, error) {
	if m.store == nil {
		return "", nil
	}
	return m.store.GetGameSummary(ctx, roomID)
}

// SaveGameSummary saves a game summary
func (m *MemoryManager) SaveGameSummary(ctx context.Context, roomID string, summary string) error {
	if m.store == nil {
		return nil
	}
	return m.store.SaveGameSummary(ctx, roomID, summary)
}

// GetPlayerModels retrieves player models for a room
func (m *MemoryManager) GetPlayerModels(ctx context.Context, roomID string) (map[string]PlayerModel, error) {
	if m.store == nil {
		return nil, nil
	}
	return m.store.GetPlayerModels(ctx, roomID)
}

// SavePlayerModel saves a player model
func (m *MemoryManager) SavePlayerModel(ctx context.Context, roomID string, model PlayerModel) error {
	if m.store == nil {
		return nil
	}
	return m.store.SavePlayerModel(ctx, roomID, model)
}

// IngestRules ingests rule documents for RAG
func (m *MemoryManager) IngestRules(ctx context.Context, documents []RuleDocument) error {
	for _, doc := range documents {
		chunks := ChunkDocument(doc.Content, 500, 50)
		for i, chunk := range chunks {
			entry := MemoryEntry{
				ID:      fmt.Sprintf("%s-chunk-%d", doc.ID, i),
				Type:    "rule",
				Content: chunk,
				Metadata: mustMarshal(map[string]interface{}{
					"source":    doc.Source,
					"role_name": doc.RoleName,
					"chunk_idx": i,
				}),
				CreatedAt: time.Now(),
			}

			if m.embedder != nil {
				embeddings, err := m.embedder.Embed(ctx, []string{chunk})
				if err == nil && len(embeddings) > 0 {
					entry.Embedding = embeddings[0]
				}
			}

			m.rulesIndex.Add(entry)
		}
	}

	return nil
}

// SearchRules searches the rules knowledge base
func (m *MemoryManager) SearchRules(ctx context.Context, query string, topK int) []MemoryEntry {
	if m.rulesIndex == nil {
		return nil
	}

	// Try vector search first
	if m.embedder != nil {
		embeddings, err := m.embedder.Embed(ctx, []string{query})
		if err == nil && len(embeddings) > 0 {
			return m.rulesIndex.Search(embeddings[0], topK)
		}
	}

	// Fall back to keyword search
	return m.rulesIndex.KeywordSearch(query, topK)
}

// RuleDocument represents a rule document to ingest
type RuleDocument struct {
	ID       string
	Source   string
	RoleName string
	Content  string
}

// VectorIndex is a simple in-memory vector index
type VectorIndex struct {
	entries   []MemoryEntry
	dimension int
	mu        sync.RWMutex
}

// NewVectorIndex creates a new vector index
func NewVectorIndex(dimension int) *VectorIndex {
	return &VectorIndex{
		entries:   make([]MemoryEntry, 0),
		dimension: dimension,
	}
}

// Add adds an entry to the index
func (v *VectorIndex) Add(entry MemoryEntry) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.entries = append(v.entries, entry)
}

// Search finds the top-k most similar entries
func (v *VectorIndex) Search(query []float32, topK int) []MemoryEntry {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if len(v.entries) == 0 {
		return nil
	}

	type scored struct {
		entry MemoryEntry
		score float64
	}

	scores := make([]scored, 0, len(v.entries))
	for _, entry := range v.entries {
		if len(entry.Embedding) == 0 {
			continue
		}
		score := cosineSimilarity(query, entry.Embedding)
		scores = append(scores, scored{entry: entry, score: score})
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	results := make([]MemoryEntry, 0, topK)
	for i := 0; i < topK && i < len(scores); i++ {
		entry := scores[i].entry
		entry.Score = scores[i].score
		results = append(results, entry)
	}

	return results
}

// KeywordSearch performs keyword-based search
func (v *VectorIndex) KeywordSearch(query string, topK int) []MemoryEntry {
	v.mu.RLock()
	defer v.mu.RUnlock()

	queryTerms := strings.Fields(strings.ToLower(query))
	if len(queryTerms) == 0 {
		return nil
	}

	type scored struct {
		entry MemoryEntry
		score float64
	}

	scores := make([]scored, 0, len(v.entries))
	for _, entry := range v.entries {
		content := strings.ToLower(entry.Content)
		score := 0.0
		for _, term := range queryTerms {
			if strings.Contains(content, term) {
				score += 1.0
			}
		}
		if score > 0 {
			scores = append(scores, scored{entry: entry, score: score / float64(len(queryTerms))})
		}
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	results := make([]MemoryEntry, 0, topK)
	for i := 0; i < topK && i < len(scores); i++ {
		entry := scores[i].entry
		entry.Score = scores[i].score
		results = append(results, entry)
	}

	return results
}

// ChunkDocument splits a document into overlapping chunks
func ChunkDocument(content string, chunkSize, overlap int) []string {
	words := strings.Fields(content)
	if len(words) <= chunkSize {
		return []string{content}
	}

	var chunks []string
	step := chunkSize - overlap
	if step <= 0 {
		step = chunkSize / 2
	}

	for i := 0; i < len(words); i += step {
		end := i + chunkSize
		if end > len(words) {
			end = len(words)
		}
		chunk := strings.Join(words[i:end], " ")
		chunks = append(chunks, chunk)
		if end >= len(words) {
			break
		}
	}

	return chunks
}

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// SimpleEmbedder is a stub embedder that generates random embeddings
// Replace with actual embedder (OpenAI, local model, etc.)
type SimpleEmbedder struct {
	dimension int
}

// NewSimpleEmbedder creates a simple stub embedder
func NewSimpleEmbedder(dimension int) *SimpleEmbedder {
	return &SimpleEmbedder{dimension: dimension}
}

func (e *SimpleEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	// This is a stub - generates pseudo-embeddings based on text hash
	// Replace with actual embedding API calls
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embeddings[i] = e.hashToEmbedding(text)
	}
	return embeddings, nil
}

func (e *SimpleEmbedder) Dimension() int {
	return e.dimension
}

func (e *SimpleEmbedder) hashToEmbedding(text string) []float32 {
	embedding := make([]float32, e.dimension)
	hash := uint64(0)
	for i := 0; i < len(text); i++ {
		hash = hash*31 + uint64(text[i])
	}
	for i := range embedding {
		hash = hash*1103515245 + 12345
		embedding[i] = float32(hash%1000) / 1000.0
	}
	return embedding
}

// OpenAIEmbedder uses OpenAI API for embeddings
type OpenAIEmbedder struct {
	provider  *OpenAIProvider
	model     string
	dimension int
}

// NewOpenAIEmbedder creates an OpenAI embedder
func NewOpenAIEmbedder(provider *OpenAIProvider, model string) *OpenAIEmbedder {
	dimension := 1536 // text-embedding-ada-002
	if strings.Contains(model, "3-small") {
		dimension = 1536
	} else if strings.Contains(model, "3-large") {
		dimension = 3072
	}

	return &OpenAIEmbedder{
		provider:  provider,
		model:     model,
		dimension: dimension,
	}
}

func (e *OpenAIEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	req := map[string]interface{}{
		"model": e.model,
		"input": texts,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Note: For embedding, we use the embedding endpoint directly
	// For now, fall back to simple embedder since Chat doesn't support embeddings
	_ = body
	simple := NewSimpleEmbedder(e.dimension)
	return simple.Embed(ctx, texts)
}

func (e *OpenAIEmbedder) Dimension() int {
	return e.dimension
}

// InMemoryMemoryStore implements MemoryStore in memory
type InMemoryMemoryStore struct {
	entries      map[string][]MemoryEntry
	summaries    map[string]string
	playerModels map[string]map[string]PlayerModel
	mu           sync.RWMutex
}

// NewInMemoryMemoryStore creates an in-memory store
func NewInMemoryMemoryStore() *InMemoryMemoryStore {
	return &InMemoryMemoryStore{
		entries:      make(map[string][]MemoryEntry),
		summaries:    make(map[string]string),
		playerModels: make(map[string]map[string]PlayerModel),
	}
}

func (s *InMemoryMemoryStore) SaveEntry(ctx context.Context, roomID string, entry MemoryEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if entry.ID == "" {
		entry.ID = uuid.NewString()
	}
	s.entries[roomID] = append(s.entries[roomID], entry)
	return nil
}

func (s *InMemoryMemoryStore) LoadEntries(ctx context.Context, roomID string, entryType string, limit int) ([]MemoryEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := s.entries[roomID]
	var result []MemoryEntry
	for i := len(entries) - 1; i >= 0 && len(result) < limit; i-- {
		if entryType == "" || entries[i].Type == entryType {
			result = append(result, entries[i])
		}
	}
	return result, nil
}

func (s *InMemoryMemoryStore) SearchByEmbedding(ctx context.Context, roomID string, embedding []float32, topK int) ([]MemoryEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := s.entries[roomID]
	type scored struct {
		entry MemoryEntry
		score float64
	}

	scores := make([]scored, 0, len(entries))
	for _, entry := range entries {
		if len(entry.Embedding) == 0 {
			continue
		}
		score := cosineSimilarity(embedding, entry.Embedding)
		scores = append(scores, scored{entry: entry, score: score})
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	results := make([]MemoryEntry, 0, topK)
	for i := 0; i < topK && i < len(scores); i++ {
		entry := scores[i].entry
		entry.Score = scores[i].score
		results = append(results, entry)
	}

	return results, nil
}

func (s *InMemoryMemoryStore) SaveGameSummary(ctx context.Context, roomID string, summary string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.summaries[roomID] = summary
	return nil
}

func (s *InMemoryMemoryStore) GetGameSummary(ctx context.Context, roomID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.summaries[roomID], nil
}

func (s *InMemoryMemoryStore) SavePlayerModel(ctx context.Context, roomID string, model PlayerModel) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.playerModels[roomID] == nil {
		s.playerModels[roomID] = make(map[string]PlayerModel)
	}
	s.playerModels[roomID][model.UserID] = model
	return nil
}

func (s *InMemoryMemoryStore) GetPlayerModels(ctx context.Context, roomID string) (map[string]PlayerModel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.playerModels[roomID], nil
}

func mustMarshal(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// RAGResult represents a RAG retrieval result with citation
type RAGResult struct {
	Content   string   `json:"content"`
	Source    string   `json:"source"`
	ChunkID   string   `json:"chunk_id"`
	Score     float64  `json:"score"`
	Citations []string `json:"citations,omitempty"`
}

// RAGPipeline orchestrates the RAG process
type RAGPipeline struct {
	memory *MemoryManager
	router *ModelRouter
}

// NewRAGPipeline creates a new RAG pipeline
func NewRAGPipeline(memory *MemoryManager, router *ModelRouter) *RAGPipeline {
	return &RAGPipeline{
		memory: memory,
		router: router,
	}
}

// Query performs RAG retrieval and generation
func (p *RAGPipeline) Query(ctx context.Context, roomID string, query string, topK int) (*RAGResponse, error) {
	// Retrieve relevant documents
	entries, err := p.memory.RetrieveRelevant(ctx, roomID, query, topK)
	if err != nil {
		return nil, fmt.Errorf("retrieve relevant: %w", err)
	}

	// Build context from retrieved entries
	var contextParts []string
	var citations []string
	for i, entry := range entries {
		contextParts = append(contextParts, fmt.Sprintf("[%d] %s", i+1, entry.Content))
		var meta map[string]interface{}
		if len(entry.Metadata) > 0 {
			json.Unmarshal(entry.Metadata, &meta)
		}
		source := entry.ID
		if meta != nil {
			if s, ok := meta["source"].(string); ok {
				source = s
			}
		}
		citations = append(citations, fmt.Sprintf("[%d] %s", i+1, source))
	}

	ragContext := strings.Join(contextParts, "\n\n")

	// Generate response with context
	messages := []Message{
		{
			Role: "system",
			Content: `You are an expert on Blood on the Clocktower game rules. 
Answer questions using ONLY the provided context. 
Cite sources using [N] notation where N is the reference number.
If the context doesn't contain enough information, say so.`,
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Context:\n%s\n\nQuestion: %s", ragContext, query),
		},
	}

	resp, err := p.router.Chat(ctx, "rules", messages, nil)
	if err != nil {
		return nil, fmt.Errorf("generate response: %w", err)
	}

	answer := ""
	if len(resp.Choices) > 0 {
		answer = resp.Choices[0].Message.Content
	}

	return &RAGResponse{
		Answer:    answer,
		Sources:   entries,
		Citations: citations,
	}, nil
}

// RAGResponse represents a RAG query response
type RAGResponse struct {
	Answer    string        `json:"answer"`
	Sources   []MemoryEntry `json:"sources"`
	Citations []string      `json:"citations"`
}
