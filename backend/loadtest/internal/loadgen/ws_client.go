package loadgen

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// WSClient handles WebSocket connections to the backend.
type WSClient struct {
	url     string
	token   string
	conn    *websocket.Conn
	mu      sync.Mutex
	closed  int32
	eventCh chan EventResponse
}

// WSMessage is a message sent/received over WebSocket.
type WSMessage struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

// WSSubscribePayload is the payload for subscribe messages.
type WSSubscribePayload struct {
	RoomID  string `json:"room_id"`
	LastSeq int64  `json:"last_seq"`
}

// WSCommandPayload is the payload for command messages.
type WSCommandPayload struct {
	CommandID      string          `json:"command_id"`
	IdempotencyKey string          `json:"idempotency_key"`
	RoomID         string          `json:"room_id"`
	Type           string          `json:"type"`
	LastSeenSeq    int64           `json:"last_seen_seq"`
	Data           json.RawMessage `json:"data,omitempty"`
}

// WSEventPayload is the payload for event messages from server.
type WSEventPayload struct {
	RoomID    string          `json:"room_id"`
	Seq       int64           `json:"seq"`
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
	ServerTS  int64           `json:"server_ts"`
}

// NewWSClient creates a new WebSocket client.
func NewWSClient(baseWSURL, token string) *WSClient {
	return &WSClient{
		url:     baseWSURL,
		token:   token,
		eventCh: make(chan EventResponse, 1000),
	}
}

// Connect establishes the WebSocket connection.
func (c *WSClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Build URL with token
	u, err := url.Parse(c.url)
	if err != nil {
		return fmt.Errorf("invalid WS URL: %w", err)
	}
	q := u.Query()
	q.Set("token", c.token)
	u.RawQuery = q.Encode()

	// Connect with dialer
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.conn = conn

	// Start reader goroutine
	go c.reader()

	return nil
}

// Close closes the WebSocket connection.
func (c *WSClient) Close() error {
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return nil // Already closed
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Subscribe subscribes to a room's events.
func (c *WSClient) Subscribe(ctx context.Context, roomID string, lastSeq int64) error {
	payload := WSSubscribePayload{
		RoomID:  roomID,
		LastSeq: lastSeq,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := WSMessage{
		Type:      "subscribe",
		RequestID: fmt.Sprintf("sub_%d", time.Now().UnixNano()),
		Payload:   payloadBytes,
	}

	return c.send(msg)
}

// SendCommand sends a game command.
func (c *WSClient) SendCommand(ctx context.Context, roomID, cmdType, idempotencyKey string, data interface{}) error {
	var dataBytes json.RawMessage
	if data != nil {
		var err error
		dataBytes, err = json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal data: %w", err)
		}
	}

	payload := WSCommandPayload{
		CommandID:      fmt.Sprintf("cmd_%d", time.Now().UnixNano()),
		IdempotencyKey: idempotencyKey,
		RoomID:         roomID,
		Type:           cmdType,
		Data:           dataBytes,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := WSMessage{
		Type:      "command",
		RequestID: fmt.Sprintf("req_%d", time.Now().UnixNano()),
		Payload:   payloadBytes,
	}

	return c.send(msg)
}

// Ping sends a ping message.
func (c *WSClient) Ping(ctx context.Context) error {
	msg := WSMessage{
		Type:      "ping",
		RequestID: fmt.Sprintf("ping_%d", time.Now().UnixNano()),
	}
	return c.send(msg)
}

// Events returns the event channel.
func (c *WSClient) Events() <-chan EventResponse {
	return c.eventCh
}

// WaitForEvents waits for n events or timeout.
func (c *WSClient) WaitForEvents(ctx context.Context, n int, timeout time.Duration) ([]EventResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var events []EventResponse
	for len(events) < n {
		select {
		case ev := <-c.eventCh:
			events = append(events, ev)
		case <-ctx.Done():
			return events, ctx.Err()
		}
	}
	return events, nil
}

// send sends a message over WebSocket.
func (c *WSClient) send(msg WSMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return fmt.Errorf("not connected")
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		return fmt.Errorf("failed to send: %w", err)
	}

	return nil
}

// reader reads messages from WebSocket.
func (c *WSClient) reader() {
	defer close(c.eventCh)

	for {
		if atomic.LoadInt32(&c.closed) == 1 {
			return
		}

		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()

		if conn == nil {
			return
		}

		var msg WSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if atomic.LoadInt32(&c.closed) == 0 {
				// Unexpected error
				// TODO: handle reconnection
			}
			return
		}

		switch msg.Type {
		case "event":
			var payload WSEventPayload
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				continue
			}
			c.eventCh <- EventResponse{
				RoomID:    payload.RoomID,
				Seq:       payload.Seq,
				EventType: payload.EventType,
				Data:      payload.Data,
				ServerTS:  payload.ServerTS,
			}
		case "pong":
			// Ignore pong
		case "error":
			// Log error
		}
	}
}
