package loadgen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPClient handles HTTP requests to the backend.
type HTTPClient struct {
	baseURL string
	client  *http.Client
}

// NewHTTPClient creates a new HTTP client.
func NewHTTPClient(baseURL string) (*HTTPClient, error) {
	return &HTTPClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}, nil
}

// RegisterResponse is the response from user registration.
type RegisterResponse struct {
	UserID string `json:"user_id"`
}

// LoginResponse is the response from user login.
type LoginResponse struct {
	Token  string `json:"token"`
	UserID string `json:"user_id"`
}

// CreateRoomResponse is the response from room creation.
type CreateRoomResponse struct {
	RoomID string `json:"room_id"`
}

// RoomResponse is the response from getting room info.
type RoomResponse struct {
	RoomID  string `json:"room_id"`
	OwnerID string `json:"owner_id"`
	Phase   string `json:"phase"`
	Players []struct {
		UserID string `json:"user_id"`
		Seat   int    `json:"seat"`
	} `json:"players"`
}

// EventResponse is a single event from the event stream.
type EventResponse struct {
	RoomID    string          `json:"room_id"`
	Seq       int64           `json:"seq"`
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
	ServerTS  int64           `json:"server_ts"`
}

// EventsResponse is the response from getting events.
type EventsResponse struct {
	Events []EventResponse `json:"events"`
}

// HealthResponse is the response from health check.
type HealthResponse struct {
	Status string `json:"status"`
}

// MetricsResponse holds raw Prometheus metrics.
type MetricsResponse struct {
	Raw string
}

// Register registers a new user.
func (c *HTTPClient) Register(ctx context.Context, email, password string) (*RegisterResponse, error) {
	body := map[string]string{
		"email":    email,
		"password": password,
	}

	resp, err := c.doJSON(ctx, "POST", "/v1/auth/register", nil, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("register failed: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	var result RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// Login logs in a user and returns a JWT token.
func (c *HTTPClient) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	body := map[string]string{
		"email":    email,
		"password": password,
	}

	resp, err := c.doJSON(ctx, "POST", "/v1/auth/login", nil, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("login failed: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	var result LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// CreateRoom creates a new room.
func (c *HTTPClient) CreateRoom(ctx context.Context, token string) (*CreateRoomResponse, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}

	resp, err := c.doJSON(ctx, "POST", "/v1/rooms", headers, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create room failed: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	var result CreateRoomResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// JoinRoom joins an existing room.
func (c *HTTPClient) JoinRoom(ctx context.Context, token, roomID string) error {
	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}

	resp, err := c.doJSON(ctx, "POST", fmt.Sprintf("/v1/rooms/%s/join", roomID), headers, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("join room failed: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// GetRoom gets room information.
func (c *HTTPClient) GetRoom(ctx context.Context, token, roomID string) (*RoomResponse, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}

	resp, err := c.doJSON(ctx, "GET", fmt.Sprintf("/v1/rooms/%s", roomID), headers, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get room failed: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	var result RoomResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetEvents gets events after a sequence number.
func (c *HTTPClient) GetEvents(ctx context.Context, token, roomID string, afterSeq int64) (*EventsResponse, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}

	path := fmt.Sprintf("/v1/rooms/%s/events?after_seq=%d", roomID, afterSeq)
	resp, err := c.doJSON(ctx, "GET", path, headers, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get events failed: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	var result EventsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetReplay gets all events up to a sequence number.
func (c *HTTPClient) GetReplay(ctx context.Context, token, roomID string, toSeq int64) (*EventsResponse, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}

	path := fmt.Sprintf("/v1/rooms/%s/replay", roomID)
	if toSeq > 0 {
		path = fmt.Sprintf("%s?to_seq=%d", path, toSeq)
	}

	resp, err := c.doJSON(ctx, "GET", path, headers, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get replay failed: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	var result EventsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// Health checks server health.
func (c *HTTPClient) Health(ctx context.Context) (*HealthResponse, error) {
	resp, err := c.doJSON(ctx, "GET", "/health", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("health check failed: %d", resp.StatusCode)
	}

	var result HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// Metrics gets Prometheus metrics.
func (c *HTTPClient) Metrics(ctx context.Context) (*MetricsResponse, error) {
	resp, err := c.doJSON(ctx, "GET", "/metrics", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metrics failed: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read metrics: %w", err)
	}

	return &MetricsResponse{Raw: string(bodyBytes)}, nil
}

// doJSON performs a JSON HTTP request.
func (c *HTTPClient) doJSON(ctx context.Context, method, path string, headers map[string]string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.client.Do(req)
}
