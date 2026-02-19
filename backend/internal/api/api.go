// Package api provides the HTTP API handlers for the Blood on the Clocktower Auto-DM server.
//
// @title Blood on the Clocktower Auto-DM API
// @version 1.0
// @description AI-powered Storyteller backend for Blood on the Clocktower game.
// @description Supports real-time WebSocket connections, event sourcing, and multi-agent AI system.
//
// @contact.name API Support
// @contact.url https://github.com/qingchang/Blood-on-the-Clocktower-auto-dm
//
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT
//
// @host localhost:8080
// @BasePath /
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter 'Bearer {token}' to authorize
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/auth"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/bot"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/engine"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/projection"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/realtime"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/room"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/store"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

type contextKey string

const userIDKey contextKey = "user_id"

type Server struct {
	Router  *chi.Mux
	store   *store.Store
	jwt     *auth.JWTManager
	roomMgr *room.RoomManager
	logger  *zap.Logger
	llmInfo *LLMInfo
	botMgr  *bot.Manager
}

// LLMInfo holds LLM provider information for the health endpoint.
type LLMInfo struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	BaseURL  string `json:"base_url"`
	Enabled  bool   `json:"enabled"`
}

func NewServer(st *store.Store, jwt *auth.JWTManager, roomMgr *room.RoomManager, wsServer *realtime.WSServer, logger *zap.Logger, opts ...ServerOption) *Server {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(corsMiddleware)

	s := &Server{
		Router:  r,
		store:   st,
		jwt:     jwt,
		roomMgr: roomMgr,
		logger:  logger,
	}

	for _, opt := range opts {
		opt(s)
	}

	// Health & Metrics
	r.Get("/health", s.health)
	r.Handle("/metrics", promhttp.Handler())
	r.Get("/v1/llm/health", s.llmHealth)

	// Swagger documentation
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// Auth endpoints
	r.Post("/v1/auth/register", s.register)
	r.Post("/v1/auth/login", s.login)
	r.Post("/v1/auth/quick", s.quickLogin)

	// Room endpoints (protected)
	r.Route("/v1/rooms", func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Post("/", s.createRoom)
		r.Post("/{room_id}/join", s.joinRoom)
		r.Get("/{room_id}/events", s.fetchEvents)
		r.Get("/{room_id}/state", s.fetchState)
		r.Get("/{room_id}/replay", s.replay)
		r.Post("/{room_id}/bots", s.addBots)
	})

	// WebSocket endpoint
	r.Handle("/ws", wsServer)
	return s
}

// corsMiddleware handles CORS for all requests.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-ID")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// health godoc
// @Summary Health check endpoint
// @Description Returns server health status
// @Tags System
// @Produce plain
// @Success 200 {string} string "ok"
// @Router /health [get]
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}

// RegisterRequest represents a user registration request.
type RegisterRequest struct {
	Email    string `json:"email" example:"user@example.com"`
	Password string `json:"password" example:"password123"`
}

// AuthResponse represents the authentication response.
type AuthResponse struct {
	Token  string `json:"token" example:"eyJhbGciOiJIUzI1NiIs..."`
	UserID string `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// register godoc
// @Summary Register a new user
// @Description Create a new user account and return JWT token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration details"
// @Success 200 {object} AuthResponse
// @Failure 400 {string} string "invalid json"
// @Failure 409 {string} string "user exists or db error"
// @Router /v1/auth/register [post]
func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "hash error", http.StatusInternalServerError)
		return
	}
	u := store.User{ID: uuid.NewString(), Email: req.Email, PasswordHash: hash, CreatedAt: time.Now().UTC()}
	if err := s.store.CreateUser(r.Context(), u); err != nil {
		http.Error(w, "user exists or db error", http.StatusConflict)
		return
	}
	token, _ := s.jwt.Generate(u.ID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{Token: token, UserID: u.ID})
}

// LoginRequest represents a login request.
type LoginRequest struct {
	Email    string `json:"email" example:"user@example.com"`
	Password string `json:"password" example:"password123"`
}

// login godoc
// @Summary User login
// @Description Authenticate user and return JWT token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} AuthResponse
// @Failure 400 {string} string "invalid json"
// @Failure 401 {string} string "invalid credentials"
// @Router /v1/auth/login [post]
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	u, err := s.store.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	if err := auth.CheckPassword(u.PasswordHash, req.Password); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	token, _ := s.jwt.Generate(u.ID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{Token: token, UserID: u.ID})
}

// QuickLoginRequest represents a quick login with just a display name.
type QuickLoginRequest struct {
	Name string `json:"name" example:"Alice"`
}

// QuickLoginResponse represents the quick login response.
type QuickLoginResponse struct {
	Token  string `json:"token" example:"eyJhbGciOiJIUzI1NiIs..."`
	UserID string `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name   string `json:"name" example:"Alice"`
}

// quickLogin godoc
// @Summary Quick login with just a display name
// @Description Create a temporary user with a display name and return JWT token (no password needed)
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body QuickLoginRequest true "Display name"
// @Success 200 {object} QuickLoginResponse
// @Failure 400 {string} string "invalid json or empty name"
// @Router /v1/auth/quick [post]
func (s *Server) quickLogin(w http.ResponseWriter, r *http.Request) {
	var req QuickLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	userID := uuid.NewString()
	uniqueEmail := userID + "@quick.local"
	u := store.User{ID: userID, Email: uniqueEmail, PasswordHash: "", CreatedAt: time.Now().UTC()}
	if err := s.store.CreateUser(r.Context(), u); err != nil {
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}
	token, _ := s.jwt.Generate(userID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(QuickLoginResponse{Token: token, UserID: userID, Name: req.Name})
}

// CreateRoomResponse represents the room creation response.
type CreateRoomResponse struct {
	RoomID string `json:"room_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// createRoom godoc
// @Summary Create a new game room
// @Description Create a new Blood on the Clocktower game room
// @Tags Rooms
// @Security BearerAuth
// @Produce json
// @Success 200 {object} CreateRoomResponse
// @Failure 401 {string} string "unauthorized"
// @Failure 500 {string} string "db error"
// @Router /v1/rooms [post]
func (s *Server) createRoom(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(string)
	rm := store.Room{ID: uuid.NewString(), CreatedBy: userID, DMUserID: userID, Status: "lobby", CreatedAt: time.Now().UTC()}
	if err := s.store.CreateRoom(r.Context(), rm); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	_ = s.store.AddRoomMember(r.Context(), store.RoomMember{RoomID: rm.ID, UserID: userID, Role: "dm", Joined: time.Now().UTC()})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CreateRoomResponse{RoomID: rm.ID})
}

// JoinRoomResponse represents the join room response.
type JoinRoomResponse struct {
	Status string `json:"status" example:"joined"`
}

// joinRoom godoc
// @Summary Join an existing game room
// @Description Join a Blood on the Clocktower game room as a player
// @Tags Rooms
// @Security BearerAuth
// @Produce json
// @Param room_id path string true "Room ID"
// @Success 200 {object} JoinRoomResponse
// @Failure 401 {string} string "unauthorized"
// @Failure 404 {string} string "room not found"
// @Router /v1/rooms/{room_id}/join [post]
func (s *Server) joinRoom(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(string)
	roomID := chi.URLParam(r, "room_id")
	if err := s.store.AddRoomMember(r.Context(), store.RoomMember{RoomID: roomID, UserID: userID, Role: "player", Joined: time.Now().UTC()}); err != nil {
		http.Error(w, "failed to join room", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(JoinRoomResponse{Status: "joined"})
}

// fetchEvents godoc
// @Summary Fetch room events
// @Description Retrieve events from a room for state synchronization (supports last_seq incremental sync)
// @Tags Events
// @Security BearerAuth
// @Produce json
// @Param room_id path string true "Room ID"
// @Param after_seq query integer false "Fetch events after this sequence number"
// @Success 200 {array} store.StoredEvent
// @Failure 401 {string} string "unauthorized"
// @Failure 403 {string} string "forbidden"
// @Router /v1/rooms/{room_id}/events [get]
func (s *Server) fetchEvents(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(string)
	roomID := chi.URLParam(r, "room_id")
	afterSeq := int64(0)
	if q := r.URL.Query().Get("after_seq"); q != "" {
		afterSeq, _ = strconv.ParseInt(q, 10, 64)
	}
	ok, _, _ := s.store.IsMember(r.Context(), roomID, userID)
	if !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	events, _ := s.store.LoadEventsAfter(r.Context(), roomID, afterSeq, 200)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

// fetchState godoc
// @Summary Fetch room state
// @Description Retrieve current game state with visibility projection based on user role
// @Tags State
// @Security BearerAuth
// @Produce json
// @Param room_id path string true "Room ID"
// @Success 200 {object} engine.State
// @Failure 401 {string} string "unauthorized"
// @Failure 403 {string} string "forbidden"
// @Failure 500 {string} string "room error"
// @Router /v1/rooms/{room_id}/state [get]
func (s *Server) fetchState(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(string)
	roomID := chi.URLParam(r, "room_id")
	ok, role, _ := s.store.IsMember(r.Context(), roomID, userID)
	if !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	ra, err := s.roomMgr.GetOrCreate(r.Context(), roomID)
	if err != nil {
		http.Error(w, "room error", http.StatusInternalServerError)
		return
	}
	state := ra.GetState()
	viewer := types.Viewer{UserID: userID, IsDM: role == "dm"}
	projected := projection.ProjectedState(state, viewer)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projected)
}

// replay godoc
// @Summary Replay game to specific point
// @Description Rebuild game state up to a specific sequence number for replay/debugging
// @Tags Events
// @Security BearerAuth
// @Produce json
// @Param room_id path string true "Room ID"
// @Param to_seq query integer false "Replay up to this sequence number"
// @Param viewer query string false "View state as specific user"
// @Success 200 {object} engine.State
// @Failure 401 {string} string "unauthorized"
// @Failure 403 {string} string "forbidden"
// @Router /v1/rooms/{room_id}/replay [get]
func (s *Server) replay(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(string)
	roomID := chi.URLParam(r, "room_id")
	toSeq := int64(0)
	if q := r.URL.Query().Get("to_seq"); q != "" {
		toSeq, _ = strconv.ParseInt(q, 10, 64)
	}
	viewerParam := r.URL.Query().Get("viewer")
	ok, role, _ := s.store.IsMember(r.Context(), roomID, userID)
	if !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	isDM := role == "dm"
	if !isDM || viewerParam == "" {
		viewerParam = userID
	}
	events, _ := s.store.LoadEventsUpTo(r.Context(), roomID, toSeq)
	state := engine.NewState(roomID)
	for _, e := range events {
		var p map[string]string
		_ = json.Unmarshal([]byte(e.PayloadJSON), &p)
		state.Reduce(engine.EventPayload{Seq: e.Seq, Type: e.EventType, Actor: e.ActorUserID, Payload: p})
	}
	viewer := types.Viewer{UserID: viewerParam, IsDM: isDM}
	projected := projection.ProjectedState(state, viewer)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projected)
}

// ServerOption configures optional Server settings.
type ServerOption func(*Server)

// WithLLMInfo sets the LLM provider info for the health endpoint.
func WithLLMInfo(info *LLMInfo) ServerOption {
	return func(s *Server) {
		s.llmInfo = info
	}
}

// WithBotManager sets the bot manager for bot endpoints.
func WithBotManager(mgr *bot.Manager) ServerOption {
	return func(s *Server) {
		s.botMgr = mgr
	}
}

// llmHealth godoc
// @Summary LLM provider health check
// @Description Returns the configured LLM provider information and connectivity status
// @Tags System
// @Produce json
// @Success 200 {object} LLMInfo
// @Router /v1/llm/health [get]
// AddBotsRequest is the request body for adding bots.
type AddBotsRequest struct {
	Count       int    `json:"count" example:"6"`
	Personality string `json:"personality,omitempty" example:"random"`
}

// AddBotsResponse is the response after adding bots.
type AddBotsResponse struct {
	BotIDs []string `json:"bot_ids"`
	Count  int      `json:"count"`
}

// addBots godoc
// @Summary Add bot players to a room
// @Description Add AI bot players to a game room for solo testing
// @Tags Rooms
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param room_id path string true "Room ID"
// @Param request body AddBotsRequest true "Bot configuration"
// @Success 200 {object} AddBotsResponse
// @Failure 400 {string} string "invalid request"
// @Failure 500 {string} string "failed to add bots"
// @Router /v1/rooms/{room_id}/bots [post]
func (s *Server) addBots(w http.ResponseWriter, r *http.Request) {
	if s.botMgr == nil {
		http.Error(w, "bot system not available", http.StatusServiceUnavailable)
		return
	}

	roomID := chi.URLParam(r, "room_id")
	var req AddBotsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Count <= 0 {
		req.Count = 6 // Default for a 7-player game (1 human + 6 bots)
	}

	ra, err := s.roomMgr.GetOrCreate(r.Context(), roomID)
	if err != nil {
		http.Error(w, "room error", http.StatusInternalServerError)
		return
	}

	botIDs, err := s.botMgr.AddBots(r.Context(), bot.AddBotsRequest{
		RoomID:      roomID,
		Count:       req.Count,
		Personality: bot.Personality(req.Personality),
	}, ra)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AddBotsResponse{BotIDs: botIDs, Count: len(botIDs)})
}

func (s *Server) llmHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.llmInfo == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "unconfigured",
			"enabled": false,
		})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "ok",
		"provider": s.llmInfo.Provider,
		"model":    s.llmInfo.Model,
		"base_url": s.llmInfo.BaseURL,
		"enabled":  s.llmInfo.Enabled,
	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if len(authHeader) < 8 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		tokenStr := authHeader[7:]
		claims, err := s.jwt.Parse(tokenStr)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
