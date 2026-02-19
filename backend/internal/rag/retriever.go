package rag

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// hashToUUID converts a sha256 hash to a valid UUID v4-like string for Qdrant.
func hashToUUID(hash [32]byte) string {
	// Format as UUID: xxxxxxxx-xxxx-4xxx-8xxx-xxxxxxxxxxxx
	hash[6] = (hash[6] & 0x0f) | 0x40 // version 4
	hash[8] = (hash[8] & 0x3f) | 0x80 // variant 1
	return fmt.Sprintf("%x-%x-%x-%x-%x", hash[0:4], hash[4:6], hash[6:8], hash[8:10], hash[10:16])
}

// Document represents a document chunk for RAG.
type Document struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
}

// RuleRetriever handles rule document retrieval.
type RuleRetriever struct {
	qdrant   *QdrantClient
	embedder EmbeddingProvider
	mu       sync.RWMutex
}

// NewRuleRetriever creates a new rule retriever.
func NewRuleRetriever(qdrant *QdrantClient, embedder EmbeddingProvider) *RuleRetriever {
	return &RuleRetriever{
		qdrant:   qdrant,
		embedder: embedder,
	}
}

// Initialize sets up the collection and indexes rule documents.
func (r *RuleRetriever) Initialize(ctx context.Context, rulesDir string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Ensure collection exists
	if err := r.qdrant.EnsureCollection(ctx, r.embedder.Dimensions()); err != nil {
		return fmt.Errorf("failed to ensure collection: %w", err)
	}

	// Check if already indexed
	count, err := r.qdrant.Count(ctx)
	if err != nil {
		return fmt.Errorf("failed to count documents: %w", err)
	}
	if count > 0 {
		return nil // Already indexed
	}

	// Load and index rule documents
	docs, err := r.loadRuleDocuments(rulesDir)
	if err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	return r.indexDocuments(ctx, docs)
}

// loadRuleDocuments loads rule documents from the rules directory.
func (r *RuleRetriever) loadRuleDocuments(rulesDir string) ([]Document, error) {
	var docs []Document

	err := filepath.Walk(rulesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") && !strings.HasSuffix(path, ".txt") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Split into chunks
		chunks := r.splitIntoChunks(string(content), path)
		docs = append(docs, chunks...)
		return nil
	})

	return docs, err
}

// splitIntoChunks splits a document into smaller chunks for better retrieval.
func (r *RuleRetriever) splitIntoChunks(content, source string) []Document {
	var docs []Document

	// Split by sections (##) or paragraphs
	sections := strings.Split(content, "\n## ")
	if len(sections) == 1 {
		// No sections, split by double newlines
		sections = strings.Split(content, "\n\n")
	}

	for i, section := range sections {
		section = strings.TrimSpace(section)
		if len(section) < 20 {
			continue
		}

		// Extract title if present
		title := ""
		lines := strings.SplitN(section, "\n", 2)
		if strings.HasPrefix(lines[0], "#") {
			title = strings.TrimLeft(lines[0], "# ")
			if len(lines) > 1 {
				section = lines[1]
			}
		}

		// Generate unique ID
		hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%s", source, i, section[:min(100, len(section))])))
		id := hashToUUID(hash)

		docs = append(docs, Document{
			ID:      id,
			Content: section,
			Metadata: map[string]interface{}{
				"source":  filepath.Base(source),
				"title":   title,
				"section": i,
			},
		})
	}

	return docs
}

// indexDocuments indexes documents into Qdrant.
func (r *RuleRetriever) indexDocuments(ctx context.Context, docs []Document) error {
	if len(docs) == 0 {
		return nil
	}

	// Batch embed
	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = doc.Content
	}

	embeddings, err := r.embedder.EmbedBatch(ctx, texts)
	if err != nil {
		return fmt.Errorf("failed to embed documents: %w", err)
	}

	// Convert to Qdrant points
	points := make([]Point, len(docs))
	for i, doc := range docs {
		payload := doc.Metadata
		payload["content"] = doc.Content
		points[i] = Point{
			ID:      doc.ID,
			Vector:  embeddings[i],
			Payload: payload,
		}
	}

	// Upsert in batches
	batchSize := 100
	for i := 0; i < len(points); i += batchSize {
		end := min(i+batchSize, len(points))
		if err := r.qdrant.Upsert(ctx, points[i:end]); err != nil {
			return fmt.Errorf("failed to upsert batch: %w", err)
		}
	}

	return nil
}

// RetrieveResult represents a retrieval result.
type RetrieveResult struct {
	Content  string                 `json:"content"`
	Score    float64                `json:"score"`
	Metadata map[string]interface{} `json:"metadata"`
}

// Retrieve searches for relevant rule documents.
func (r *RuleRetriever) Retrieve(ctx context.Context, query string, limit int) ([]RetrieveResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Embed query
	queryVec, err := r.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Search
	results, err := r.qdrant.Search(ctx, queryVec, limit, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	// Convert results
	retrieved := make([]RetrieveResult, len(results))
	for i, r := range results {
		content := ""
		if c, ok := r.Payload["content"].(string); ok {
			content = c
		}
		delete(r.Payload, "content")
		retrieved[i] = RetrieveResult{
			Content:  content,
			Score:    r.Score,
			Metadata: r.Payload,
		}
	}

	return retrieved, nil
}

// RetrieveWithFilter searches with metadata filters.
func (r *RuleRetriever) RetrieveWithFilter(ctx context.Context, query string, limit int, filter map[string]interface{}) ([]RetrieveResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	queryVec, err := r.embedder.Embed(ctx, query)
	if err != nil {
		return nil, err
	}

	// Build Qdrant filter
	qdrantFilter := buildQdrantFilter(filter)

	results, err := r.qdrant.Search(ctx, queryVec, limit, qdrantFilter)
	if err != nil {
		return nil, err
	}

	retrieved := make([]RetrieveResult, len(results))
	for i, r := range results {
		content := ""
		if c, ok := r.Payload["content"].(string); ok {
			content = c
		}
		delete(r.Payload, "content")
		retrieved[i] = RetrieveResult{
			Content:  content,
			Score:    r.Score,
			Metadata: r.Payload,
		}
	}

	return retrieved, nil
}

// buildQdrantFilter converts a simple filter map to Qdrant filter format.
func buildQdrantFilter(filter map[string]interface{}) map[string]interface{} {
	if len(filter) == 0 {
		return nil
	}

	conditions := make([]map[string]interface{}, 0, len(filter))
	for key, value := range filter {
		conditions = append(conditions, map[string]interface{}{
			"key":   key,
			"match": map[string]interface{}{"value": value},
		})
	}

	return map[string]interface{}{
		"must": conditions,
	}
}

// IndexRoleRules indexes role-specific rules.
func (r *RuleRetriever) IndexRoleRules(ctx context.Context, roleID, roleName, rules string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	hash := sha256.Sum256([]byte(roleID + rules))
	id := hashToUUID(hash)

	embedding, err := r.embedder.Embed(ctx, rules)
	if err != nil {
		return err
	}

	point := Point{
		ID:     id,
		Vector: embedding,
		Payload: map[string]interface{}{
			"content":   rules,
			"type":      "role",
			"role_id":   roleID,
			"role_name": roleName,
		},
	}

	return r.qdrant.Upsert(ctx, []Point{point})
}

// GetRoleRules retrieves rules for a specific role.
func (r *RuleRetriever) GetRoleRules(ctx context.Context, roleID string) ([]RetrieveResult, error) {
	return r.RetrieveWithFilter(ctx, roleID, 5, map[string]interface{}{
		"role_id": roleID,
	})
}
