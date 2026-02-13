package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

// GeminiEmbedding uses Google Gemini API for embeddings.
type GeminiEmbedding struct {
	apiKey     string
	model      string
	dimensions int
	httpClient *http.Client
}

// GeminiEmbeddingConfig holds configuration for Gemini embeddings.
type GeminiEmbeddingConfig struct {
	APIKey     string
	Model      string
	HTTPSProxy string
}

// NewGeminiEmbedding creates a new Gemini embedding provider.
func NewGeminiEmbedding(cfg GeminiEmbeddingConfig) *GeminiEmbedding {
	if cfg.Model == "" {
		cfg.Model = "embedding-001"
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	if cfg.HTTPSProxy != "" {
		if u, err := url.Parse(cfg.HTTPSProxy); err == nil {
			httpClient.Transport = &http.Transport{
				Proxy: http.ProxyURL(u),
			}
		}
	}

	return &GeminiEmbedding{
		apiKey:     cfg.APIKey,
		model:      cfg.Model,
		dimensions: 768, // text-embedding-004 defaults to 768
		httpClient: httpClient,
	}
}

func (e *GeminiEmbedding) Dimensions() int {
	return e.dimensions
}

func (e *GeminiEmbedding) Embed(ctx context.Context, text string) ([]float64, error) {
	batch, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(batch) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}
	return batch[0], nil
}

type geminiEmbedRequest struct {
	Requests []geminiEmbedContentRequest `json:"requests"`
}

type geminiEmbedContentRequest struct {
	Model   string        `json:"model"`
	Content geminiContent `json:"content"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiEmbedResponse struct {
	Embeddings []geminiEmbeddingValues `json:"embeddings"`
}

type geminiEmbeddingValues struct {
	Values []float64 `json:"values"`
}

func (e *GeminiEmbedding) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:batchEmbedContents?key=%s", e.model, e.apiKey)

	reqRequests := make([]geminiEmbedContentRequest, len(texts))
	for i, text := range texts {
		reqRequests[i] = geminiEmbedContentRequest{
			Model: "models/" + e.model,
			Content: geminiContent{
				Parts: []geminiPart{
					{Text: text},
				},
			},
		}
	}

	reqBody := geminiEmbedRequest{
		Requests: reqRequests,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	// DEBUG: Print details
	fmt.Printf("[Gemini Embed] URL: %s\n", url)
	// fmt.Printf("[Gemini Embed] Body: %s\n", string(jsonBody))

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("[Gemini Embed] Error Body: %s\n", string(bodyBytes))
		return nil, fmt.Errorf("gemini embedding failed: %s - %s", resp.Status, string(bodyBytes))
	}

	var geminiResp geminiEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, err
	}

	results := make([][]float64, len(geminiResp.Embeddings))
	for i, emb := range geminiResp.Embeddings {
		results[i] = emb.Values
	}

	return results, nil
}
