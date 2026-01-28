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

// QdrantClient handles communication with Qdrant vector database.
type QdrantClient struct {
	host       string
	port       int
	collection string
	httpClient *http.Client
}

// NewQdrantClient creates a new Qdrant client.
func NewQdrantClient(host string, port int, collection string) *QdrantClient {
	return &QdrantClient{
		host:       host,
		port:       port,
		collection: collection,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Point represents a Qdrant point.
type Point struct {
	ID      string                 `json:"id"`
	Vector  []float64              `json:"vector"`
	Payload map[string]interface{} `json:"payload"`
}

// SearchResult represents a search result from Qdrant.
type SearchResult struct {
	ID      string                 `json:"id"`
	Score   float64                `json:"score"`
	Payload map[string]interface{} `json:"payload"`
}

// EnsureCollection creates the collection if it doesn't exist.
func (c *QdrantClient) EnsureCollection(ctx context.Context, vectorSize int) error {
	url := fmt.Sprintf("http://%s:%d/collections/%s", c.host, c.port, c.collection)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	createReq := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     vectorSize,
			"distance": "Cosine",
		},
	}

	body, _ := json.Marshal(createReq)
	req, err = http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err = c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create collection: %s", string(respBody))
	}

	return nil
}

// Upsert inserts or updates points in the collection.
func (c *QdrantClient) Upsert(ctx context.Context, points []Point) error {
	url := fmt.Sprintf("http://%s:%d/collections/%s/points", c.host, c.port, c.collection)

	reqBody := map[string]interface{}{"points": points}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upsert failed: %s", string(respBody))
	}

	return nil
}

// Search performs a vector similarity search.
func (c *QdrantClient) Search(ctx context.Context, vector []float64, limit int, filter map[string]interface{}) ([]SearchResult, error) {
	url := fmt.Sprintf("http://%s:%d/collections/%s/points/search", c.host, c.port, c.collection)

	reqBody := map[string]interface{}{
		"vector":       vector,
		"limit":        limit,
		"with_payload": true,
	}
	if filter != nil {
		reqBody["filter"] = filter
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed: %s", string(respBody))
	}

	var result struct {
		Result []struct {
			ID      string                 `json:"id"`
			Score   float64                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	results := make([]SearchResult, len(result.Result))
	for i, r := range result.Result {
		results[i] = SearchResult{ID: r.ID, Score: r.Score, Payload: r.Payload}
	}

	return results, nil
}

// Delete removes points by IDs.
func (c *QdrantClient) Delete(ctx context.Context, ids []string) error {
	url := fmt.Sprintf("http://%s:%d/collections/%s/points/delete", c.host, c.port, c.collection)

	reqBody := map[string]interface{}{"points": ids}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed: %s", string(respBody))
	}

	return nil
}

// Count returns the number of points in the collection.
func (c *QdrantClient) Count(ctx context.Context) (int64, error) {
	url := fmt.Sprintf("http://%s:%d/collections/%s", c.host, c.port, c.collection)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Result struct {
			PointsCount int64 `json:"points_count"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	return result.Result.PointsCount, nil
}
