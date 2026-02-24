package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/auth"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/observability"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/projection"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/room"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/store"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

type WSMessage struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}

type SubscribePayload struct {
	RoomID  string `json:"room_id"`
	LastSeq int64  `json:"last_seq"`
}

type CommandPayload struct {
	CommandID      string          `json:"command_id"`
	IdempotencyKey string          `json:"idempotency_key"`
	RoomID         string          `json:"room_id"`
	Type           string          `json:"type"`
	LastSeenSeq    int64           `json:"last_seen_seq"`
	Data           json.RawMessage `json:"data"`
}

type WSServer struct {
	upgrader websocket.Upgrader
	jwt      *auth.JWTManager
	store    *store.Store
	roomMgr  *room.RoomManager
	logger   *zap.Logger
	metrics  *observability.Metrics
}

func NewWSServer(jwt *auth.JWTManager, st *store.Store, roomMgr *room.RoomManager, logger *zap.Logger, metrics *observability.Metrics) *WSServer {
	return &WSServer{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		jwt:     jwt,
		store:   st,
		roomMgr: roomMgr,
		logger:  logger,
		metrics: metrics,
	}
}

func (ws *WSServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}
	claims, err := ws.jwt.Parse(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		ws.logger.Warn("upgrade failed", zap.Error(err))
		return
	}
	sessionID := uuid.NewString()
	session := &Session{
		id:      sessionID,
		userID:  claims.UserID,
		conn:    conn,
		store:   ws.store,
		roomMgr: ws.roomMgr,
		logger:  ws.logger.With(zap.String("session_id", sessionID), zap.String("user_id", claims.UserID)), // FIX-11: Use same session ID
		metrics: ws.metrics,
		send:    make(chan []byte, 64),
		limiter: NewTokenBucket(10, 2),
	}
	ws.metrics.ActiveConnections.Inc()
	go session.writePump()
	session.readPump()
	ws.metrics.ActiveConnections.Dec()
}

type Session struct {
	id      string
	userID  string
	conn    *websocket.Conn
	store   *store.Store
	roomMgr *room.RoomManager
	logger  *zap.Logger
	metrics *observability.Metrics
	send    chan []byte
	subRoom string
	subID   string
	limiter *TokenBucket
	mu      sync.Mutex
}

func (s *Session) readPump() {
	defer func() {
		if s.subID != "" {
			ra, _ := s.roomMgr.GetOrCreate(context.Background(), s.subRoom)
			if ra != nil {
				ra.Unsubscribe(s.subID)
			}
		}
		s.conn.Close()
	}()
	s.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	s.conn.SetPongHandler(func(string) error {
		s.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		_, data, err := s.conn.ReadMessage()
		if err != nil {
			break
		}
		s.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		if !s.limiter.Allow() {
			s.sendError("", "rate_limited", "too many requests")
			continue
		}
		var msg WSMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			s.sendError("", "bad_request", "invalid json")
			continue
		}
		s.handleMessage(msg)
	}
}

func (s *Session) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		s.conn.Close()
	}()
	for {
		select {
		case data, ok := <-s.send:
			if !ok {
				return
			}
			s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := s.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-ticker.C:
			s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := s.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (s *Session) handleMessage(msg WSMessage) {
	switch msg.Type {
	case "ping":
		// FIX-10: Echo back the client's payload (contains timestamp for latency calculation)
		pongPayload := msg.Payload
		if len(pongPayload) == 0 {
			pongPayload = json.RawMessage("{}")
		}
		s.sendRaw(WSMessage{Type: "pong", RequestID: msg.RequestID, Payload: pongPayload})
	case "subscribe":
		var payload SubscribePayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			s.sendError(msg.RequestID, "bad_request", "invalid subscribe payload")
			return
		}
		s.handleSubscribe(msg.RequestID, payload)
	case "command":
		var payload CommandPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			s.sendError(msg.RequestID, "bad_request", "invalid command payload")
			return
		}
		s.handleCommand(msg.RequestID, payload)
	default:
		s.sendError(msg.RequestID, "bad_request", "unknown message type")
	}
}

func (s *Session) handleSubscribe(reqID string, payload SubscribePayload) {
	ctx := context.Background()
	ok, role, err := s.store.IsMember(ctx, payload.RoomID, s.userID)
	if err != nil || !ok {
		s.sendError(reqID, "forbidden", "not a member of room")
		return
	}
	ra, err := s.roomMgr.GetOrCreate(ctx, payload.RoomID)
	if err != nil {
		s.sendError(reqID, "internal", "cannot load room")
		return
	}
	s.subRoom = payload.RoomID
	s.subID = s.id
	isDM := role == "dm"
	ra.Subscribe(s.subID, &room.Subscriber{
		UserID: s.userID,
		IsDM:   isDM,
		Send: func(pe types.ProjectedEvent) {
			b, _ := json.Marshal(WSMessage{Type: "event", Payload: mustMarshal(pe)})
			select {
			case s.send <- b:
			default:
			}
		},
	})
	events, _ := s.store.LoadEventsAfter(ctx, payload.RoomID, payload.LastSeq, 200)
	state := ra.GetState()
	viewer := types.Viewer{UserID: s.userID, IsDM: isDM}
	for _, e := range events {
		ev := types.Event{
			RoomID:            e.RoomID,
			Seq:               e.Seq,
			EventID:           e.EventID,
			EventType:         e.EventType,
			ActorUserID:       e.ActorUserID,
			CausationCommand:  e.CausationCommand,
			Payload:           json.RawMessage(e.PayloadJSON),
			ServerTimestampMs: e.ServerTime.UnixMilli(),
		}
		pe := projection.Project(ev, state, viewer)
		if pe == nil {
			continue
		}
		b, _ := json.Marshal(WSMessage{Type: "event", Payload: mustMarshal(pe)})
		s.send <- b
		s.metrics.ResyncEvents.Inc()
	}
	s.sendRaw(WSMessage{Type: "subscribed", RequestID: reqID, Payload: json.RawMessage(`{"status":"ok"}`)})
}

func (s *Session) handleCommand(reqID string, payload CommandPayload) {
	ctx := context.Background()
	ok, _, err := s.store.IsMember(ctx, payload.RoomID, s.userID)
	if err != nil || !ok {
		s.sendError(reqID, "forbidden", "not a member")
		return
	}
	ra, err := s.roomMgr.GetOrCreate(ctx, payload.RoomID)
	if err != nil {
		s.sendError(reqID, "internal", "cannot load room")
		return
	}
	commandID := payload.CommandID
	if commandID == "" {
		commandID = uuid.NewString()
	}
	idempotencyKey := payload.IdempotencyKey
	if idempotencyKey == "" {
		idempotencyKey = commandID
	}
	cmd := types.CommandEnvelope{
		CommandID:      commandID,
		IdempotencyKey: idempotencyKey,
		RoomID:         payload.RoomID,
		Type:           payload.Type,
		LastSeenSeq:    payload.LastSeenSeq,
		ActorUserID:    s.userID,
		Payload:        payload.Data,
	}
	resp := ra.Dispatch(cmd)
	if resp.Err != nil {
		s.sendCommandResult(reqID, &types.CommandResult{CommandID: commandID, Status: "rejected", Reason: resp.Err.Error()})
		return
	}
	s.sendCommandResult(reqID, resp.Result)
}

func (s *Session) sendError(reqID, code, message string) {
	payload := map[string]string{"code": code, "message": message}
	b, _ := json.Marshal(WSMessage{Type: "error", RequestID: reqID, Payload: mustMarshal(payload)})
	s.send <- b
}

func (s *Session) sendCommandResult(reqID string, res *types.CommandResult) {
	b, _ := json.Marshal(WSMessage{Type: "command_result", RequestID: reqID, Payload: mustMarshal(res)})
	s.send <- b
}

func (s *Session) sendRaw(msg WSMessage) {
	b, _ := json.Marshal(msg)
	s.send <- b
}

func mustMarshal(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

type TokenBucket struct {
	mu       sync.Mutex
	tokens   float64
	capacity float64
	rate     float64
	lastTime time.Time
}

func NewTokenBucket(capacity, rate float64) *TokenBucket {
	return &TokenBucket{tokens: capacity, capacity: capacity, rate: rate, lastTime: time.Now()}
}

func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(tb.lastTime).Seconds()
	tb.tokens += elapsed * tb.rate
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}
	tb.lastTime = now
	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}
