package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// EmbeddingProvider generates embeddings from text.
type EmbeddingProvider interface {
	Embed(ctx context.Context, text string) ([]float64, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float64, error)
	Dimensions() int
}

// OpenAIEmbedding uses OpenAI API for embeddings.
type OpenAIEmbedding struct {
	apiKey     string
	baseURL    string
	model      string
	dimensions int
	httpClient *http.Client
}

// OpenAIEmbeddingConfig configures the OpenAI embedding provider.
type OpenAIEmbeddingConfig struct {
	APIKey     string
	BaseURL    string // For OpenAI-compatible APIs
	Model      string
	Dimensions int
}

// NewOpenAIEmbedding creates a new OpenAI embedding provider.
func NewOpenAIEmbedding(cfg OpenAIEmbeddingConfig) *OpenAIEmbedding {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.openai.com/v1"
	}
	if cfg.Model == "" {
		cfg.Model = "text-embedding-3-small"
	}
	if cfg.Dimensions == 0 {
		cfg.Dimensions = 1536
	}

	return &OpenAIEmbedding{
		apiKey:     cfg.APIKey,
		baseURL:    cfg.BaseURL,
		model:      cfg.Model,
		dimensions: cfg.Dimensions,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Embed generates an embedding for a single text.
func (e *OpenAIEmbedding) Embed(ctx context.Context, text string) ([]float64, error) {
	embeddings, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}
	return embeddings[0], nil
}

// EmbedBatch generates embeddings for multiple texts.
func (e *OpenAIEmbedding) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	reqBody := map[string]interface{}{
		"input": texts,
		"model": e.model,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/embeddings", e.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", e.apiKey))

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding request failed: %s", string(respBody))
	}

	var result struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	embeddings := make([][]float64, len(result.Data))
	for _, d := range result.Data {
		embeddings[d.Index] = d.Embedding
	}

	return embeddings, nil
}

// Dimensions returns the embedding dimension size.
func (e *OpenAIEmbedding) Dimensions() int {
	return e.dimensions
}

// LocalEmbedding provides a simple local embedding (for testing).
// In production, use a proper embedding model.
type LocalEmbedding struct {
	dimensions int
}

// NewLocalEmbedding creates a local embedding provider.
func NewLocalEmbedding(dimensions int) *LocalEmbedding {
	return &LocalEmbedding{dimensions: dimensions}
}

// Embed generates a simple hash-based embedding (for testing only).
func (e *LocalEmbedding) Embed(ctx context.Context, text string) ([]float64, error) {
	embeddings, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return embeddings[0], nil
}

// EmbedBatch generates embeddings for multiple texts.
func (e *LocalEmbedding) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	embeddings := make([][]float64, len(texts))
	for i, text := range texts {
		embedding := make([]float64, e.dimensions)
		// Simple hash-based embedding for testing
		for j, char := range text {
			idx := j % e.dimensions
			embedding[idx] += float64(char) / 1000.0
		}
		// Normalize
		var sum float64
		for _, v := range embedding {
			sum += v * v
		}
		if sum > 0 {
			norm := 1.0 / sum
			for j := range embedding {
				embedding[j] *= norm
			}
		}
		embeddings[i] = embedding
	}
	return embeddings, nil
}

// Dimensions returns the embedding dimension size.
func (e *LocalEmbedding) Dimensions() int {
	return e.dimensions
}
