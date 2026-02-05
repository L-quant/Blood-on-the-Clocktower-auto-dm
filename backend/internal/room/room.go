package room

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/engine"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/observability"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/projection"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/store"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

type CommandRequest struct {
	Cmd      types.CommandEnvelope
	Response chan CommandResponse
}

type CommandResponse struct {
	Result *types.CommandResult
	Err    error
}

type Subscriber struct {
	UserID string
	IsDM   bool
	Send   func(types.ProjectedEvent)
}

type RoomActor struct {
	RoomID   string
	ctx      context.Context
	onCrash  func(roomID string)
	subsMu   sync.RWMutex
	stateMu  sync.RWMutex
	state    engine.State
	store    *store.Store
	logger   *zap.Logger
	metrics  *observability.Metrics
	cmdCh    chan CommandRequest
	subs     map[string]*Subscriber
	snapshot int64
	autoDM   *agent.AutoDM
}

func NewRoomActor(loadCtx context.Context, loopCtx context.Context, roomID string, st *store.Store, logger *zap.Logger, metrics *observability.Metrics, snapshotInterval int64, autoDM *agent.AutoDM, onCrash func(roomID string)) (*RoomActor, error) {
	if loopCtx == nil {
		loopCtx = context.Background()
	}
	if loadCtx == nil {
		loadCtx = context.Background()
	}
	ra := &RoomActor{
		RoomID:   roomID,
		ctx:      loopCtx,
		onCrash:  onCrash,
		store:    st,
		logger:   logger,
		metrics:  metrics,
		cmdCh:    make(chan CommandRequest, 256),
		subs:     make(map[string]*Subscriber),
		snapshot: snapshotInterval,
		autoDM:   autoDM,
	}
	if err := ra.loadState(loadCtx); err != nil {
		return nil, err
	}

	go ra.loop(loopCtx)
	return ra, nil
}

func (ra *RoomActor) loadState(ctx context.Context) error {
	ra.stateMu.Lock()
	defer ra.stateMu.Unlock()

	snap, err := ra.store.GetLatestSnapshot(ctx, ra.RoomID)
	if err != nil {
		return err
	}
	if snap != nil {
		s, err := engine.UnmarshalState(snap.StateJSON)
		if err != nil {
			return err
		}
		ra.state = s
	} else {
		ra.state = engine.NewState(ra.RoomID)
	}
	afterSeq := ra.state.LastSeq
	events, err := ra.store.LoadEventsAfter(ctx, ra.RoomID, afterSeq, 0)
	if err != nil {
		return err
	}
	for _, e := range events {
		payload := toEventPayload(e)
		ra.state.Reduce(payload)
	}
	return nil
}

func toEventPayload(e store.StoredEvent) engine.EventPayload {
	var p map[string]string
	_ = json.Unmarshal([]byte(e.PayloadJSON), &p)
	return engine.EventPayload{
		Seq:     e.Seq,
		Type:    e.EventType,
		Actor:   e.ActorUserID,
		Payload: p,
	}
}

func (ra *RoomActor) loop(ctx context.Context) {
	defer func() {
		if recovered := recover(); recovered != nil {
			ra.logger.Error("room actor crashed",
				zap.String("room_id", ra.RoomID),
				zap.Any("panic", recovered),
				zap.ByteString("stack", debug.Stack()))
			if ra.onCrash != nil {
				go ra.onCrash(ra.RoomID)
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case req := <-ra.cmdCh:
			result, err, fatal := ra.executeCommand(ctx, req.Cmd)
			req.Response <- CommandResponse{Result: result, Err: err}
			if fatal {
				panic(err)
			}
		}
	}
}

func (ra *RoomActor) executeCommand(ctx context.Context, cmd types.CommandEnvelope) (result *types.CommandResult, err error, fatal bool) {
	defer func() {
		if recovered := recover(); recovered != nil {
			ra.logger.Error("room actor command panic",
				zap.String("room_id", ra.RoomID),
				zap.String("command_type", cmd.Type),
				zap.Any("panic", recovered),
				zap.ByteString("stack", debug.Stack()))
			err = fmt.Errorf("room actor panic: %v", recovered)
			fatal = true
		}
	}()
	result, err = ra.handleCommand(ctx, cmd)
	return result, err, false
}

func (ra *RoomActor) handleCommand(ctx context.Context, cmd types.CommandEnvelope) (*types.CommandResult, error) {
	if cmd.RoomID != ra.RoomID {
		return nil, fmt.Errorf("room mismatch: actor=%s command=%s", ra.RoomID, cmd.RoomID)
	}

	dedup, err := ra.store.GetDedupRecord(ctx, cmd.RoomID, cmd.ActorUserID, cmd.IdempotencyKey, cmd.Type)
	if err != nil {
		return nil, err
	}
	if dedup != nil {
		ra.metrics.DedupHitTotal.Inc()
		var result types.CommandResult
		_ = json.Unmarshal([]byte(dedup.ResultJSON), &result)
		return &result, nil
	}
	currentState := ra.GetState()

	events, result, err := engine.HandleCommand(currentState, cmd)
	if err != nil {
		ra.metrics.CommandReject.WithLabelValues("engine").Inc()
		return nil, err
	}
	storedEvents := make([]store.StoredEvent, len(events))
	for i, e := range events {
		storedEvents[i] = store.StoredEvent{
			RoomID:           e.RoomID,
			EventID:          e.EventID,
			EventType:        e.EventType,
			ActorUserID:      e.ActorUserID,
			CausationCommand: e.CausationCommand,
			PayloadJSON:      string(e.Payload),
			ServerTime:       time.Now().UTC(),
		}
	}
	dedupRec := store.DedupRecord{
		RoomID:         cmd.RoomID,
		ActorUserID:    cmd.ActorUserID,
		IdempotencyKey: cmd.IdempotencyKey,
		CommandType:    cmd.Type,
		CommandID:      cmd.CommandID,
		Status:         result.Status,
		ResultJSON:     "",
		CreatedAt:      time.Now().UTC(),
	}
	nextState := currentState.Copy()
	for i := range storedEvents {
		storedEvents[i].Seq = currentState.LastSeq + int64(i+1)
		payload := toEventPayload(storedEvents[i])
		nextState.Reduce(payload)
	}

	if len(storedEvents) > 0 {
		result.AppliedSeqFrom = storedEvents[0].Seq
		result.AppliedSeqTo = storedEvents[len(storedEvents)-1].Seq
	}
	rj, _ := json.Marshal(result)
	dedupRec.ResultJSON = string(rj)

	var snap *store.Snapshot
	if len(storedEvents) > 0 && ra.snapshot > 0 && nextState.LastSeq > 0 && nextState.LastSeq%ra.snapshot == 0 {
		stateJSON, _ := engine.MarshalState(nextState)
		snap = &store.Snapshot{
			RoomID:    ra.RoomID,
			LastSeq:   nextState.LastSeq,
			StateJSON: stateJSON,
			CreatedAt: time.Now().UTC(),
		}
	}
	if err := ra.store.AppendEvents(ctx, ra.RoomID, storedEvents, &dedupRec, snap); err != nil {
		return nil, err
	}

	ra.stateMu.Lock()
	ra.state = nextState
	stateSnapshot := ra.state.Copy()
	ra.stateMu.Unlock()

	ra.broadcast(ctx, storedEvents, stateSnapshot)
	return result, nil
}

func (ra *RoomActor) broadcast(ctx context.Context, events []store.StoredEvent, state engine.State) {
	ra.subsMu.RLock()
	defer ra.subsMu.RUnlock()

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

		// Notify subscribers (WebSocket clients)
		for _, sub := range ra.subs {
			viewer := types.Viewer{UserID: sub.UserID, IsDM: sub.IsDM}
			projected := projection.Project(ev, state, viewer)
			if projected != nil {
				sub.Send(*projected)
			}
		}

		// Notify AutoDM to respond to game events
		if ra.autoDM != nil && ra.autoDM.Enabled() {
			go ra.autoDM.OnEvent(ctx, ev, state)
		}
	}
}

func (ra *RoomActor) Subscribe(id string, s *Subscriber) {
	ra.subsMu.Lock()
	defer ra.subsMu.Unlock()
	ra.subs[id] = s
}

func (ra *RoomActor) Unsubscribe(id string) {
	ra.subsMu.Lock()
	defer ra.subsMu.Unlock()
	delete(ra.subs, id)
}

func (ra *RoomActor) Dispatch(cmd types.CommandEnvelope) CommandResponse {
	ch := make(chan CommandResponse, 1)
	select {
	case ra.cmdCh <- CommandRequest{Cmd: cmd, Response: ch}:
	case <-ra.ctx.Done():
		return CommandResponse{Err: fmt.Errorf("room actor stopped")}
	}

	select {
	case resp := <-ch:
		return resp
	case <-ra.ctx.Done():
		return CommandResponse{Err: fmt.Errorf("room actor stopped")}
	}
}

// DispatchAsync implements the agent.CommandDispatcher interface.
// It dispatches commands asynchronously without blocking.
func (ra *RoomActor) DispatchAsync(cmd types.CommandEnvelope) error {
	resp := ra.Dispatch(cmd)
	return resp.Err
}

func (ra *RoomActor) GetState() engine.State {
	ra.stateMu.RLock()
	defer ra.stateMu.RUnlock()
	return ra.state.Copy()
}

type RoomManager struct {
	mu       sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
	actors   map[string]*RoomActor
	store    *store.Store
	logger   *zap.Logger
	metrics  *observability.Metrics
	snapshot int64
	autoDM   *agent.AutoDM
}

func NewRoomManager(ctx context.Context, st *store.Store, logger *zap.Logger, metrics *observability.Metrics, snapshotInterval int64, autoDM *agent.AutoDM) *RoomManager {
	if ctx == nil {
		ctx = context.Background()
	}
	actorCtx, cancel := context.WithCancel(ctx)
	return &RoomManager{
		ctx:      actorCtx,
		cancel:   cancel,
		actors:   make(map[string]*RoomActor),
		store:    st,
		logger:   logger,
		metrics:  metrics,
		snapshot: snapshotInterval,
		autoDM:   autoDM,
	}
}

func (m *RoomManager) Close() {
	m.cancel()
}

func (m *RoomManager) GetOrCreate(ctx context.Context, roomID string) (*RoomActor, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ra, ok := m.actors[roomID]; ok {
		return ra, nil
	}
	ra, err := NewRoomActor(ctx, m.ctx, roomID, m.store, m.logger, m.metrics, m.snapshot, m.autoDM, m.handleActorCrash)
	if err != nil {
		return nil, err
	}
	m.actors[roomID] = ra
	return ra, nil
}

func (m *RoomManager) handleActorCrash(roomID string) {
	reloadCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ra, err := NewRoomActor(reloadCtx, m.ctx, roomID, m.store, m.logger, m.metrics, m.snapshot, m.autoDM, m.handleActorCrash)
	if err != nil {
		m.logger.Error("failed to restart room actor", zap.String("room_id", roomID), zap.Error(err))
		return
	}

	m.mu.Lock()
	m.actors[roomID] = ra
	m.mu.Unlock()

	m.logger.Warn("room actor restarted", zap.String("room_id", roomID))
}

// DispatchAsync routes a command to the correct room actor by room ID.
func (m *RoomManager) DispatchAsync(cmd types.CommandEnvelope) error {
	ra, err := m.GetOrCreate(context.Background(), cmd.RoomID)
	if err != nil {
		return err
	}
	resp := ra.Dispatch(cmd)
	return resp.Err
}
