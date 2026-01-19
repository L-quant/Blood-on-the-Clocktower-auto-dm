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
	"go.uber.org/zap"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/auth"
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
}

func NewServer(st *store.Store, jwt *auth.JWTManager, roomMgr *room.RoomManager, wsServer *realtime.WSServer, logger *zap.Logger) *Server {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	s := &Server{
		Router:  r,
		store:   st,
		jwt:     jwt,
		roomMgr: roomMgr,
		logger:  logger,
	}

	r.Get("/health", s.health)
	r.Handle("/metrics", promhttp.Handler())
	r.Post("/v1/auth/register", s.register)
	r.Post("/v1/auth/login", s.login)
	r.Route("/v1/rooms", func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Post("/", s.createRoom)
		r.Post("/{room_id}/join", s.joinRoom)
		r.Get("/{room_id}/events", s.fetchEvents)
		r.Get("/{room_id}/state", s.fetchState)
		r.Get("/{room_id}/replay", s.replay)
	})
	r.Handle("/ws", wsServer)
	return s
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
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
	json.NewEncoder(w).Encode(map[string]string{"token": token, "user_id": u.ID})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
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
	json.NewEncoder(w).Encode(map[string]string{"token": token, "user_id": u.ID})
}

func (s *Server) createRoom(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(string)
	rm := store.Room{ID: uuid.NewString(), CreatedBy: userID, DMUserID: userID, Status: "lobby", CreatedAt: time.Now().UTC()}
	if err := s.store.CreateRoom(r.Context(), rm); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	_ = s.store.AddRoomMember(r.Context(), store.RoomMember{RoomID: rm.ID, UserID: userID, Role: "dm", Joined: time.Now().UTC()})
	json.NewEncoder(w).Encode(map[string]string{"room_id": rm.ID})
}

func (s *Server) joinRoom(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(string)
	roomID := chi.URLParam(r, "room_id")
	_ = s.store.AddRoomMember(r.Context(), store.RoomMember{RoomID: roomID, UserID: userID, Role: "player", Joined: time.Now().UTC()})
	json.NewEncoder(w).Encode(map[string]string{"status": "joined"})
}

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
	json.NewEncoder(w).Encode(events)
}

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
	json.NewEncoder(w).Encode(projected)
}

func (s *Server) replay(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(string)
	roomID := chi.URLParam(r, "room_id")
	toSeq := int64(0)
	if q := r.URL.Query().Get("to_seq"); q != "" {
		toSeq, _ = strconv.ParseInt(q, 10, 64)
	}
	viewerParam := r.URL.Query().Get("viewer")
	if viewerParam == "" {
		viewerParam = userID
	}
	ok, role, _ := s.store.IsMember(r.Context(), roomID, userID)
	if !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	isDM := role == "dm"
	events, _ := s.store.LoadEventsAfter(r.Context(), roomID, 0, int(toSeq))
	state := engine.NewState(roomID)
	for _, e := range events {
		var p map[string]string
		_ = json.Unmarshal([]byte(e.PayloadJSON), &p)
		state.Reduce(engine.EventPayload{Seq: e.Seq, Type: e.EventType, Actor: e.ActorUserID, Payload: p})
	}
	viewer := types.Viewer{UserID: viewerParam, IsDM: isDM}
	projected := projection.ProjectedState(state, viewer)
	json.NewEncoder(w).Encode(projected)
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
