package room

import (
	"context"
	"encoding/json"
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
	mu       sync.Mutex
	state    engine.State
	store    *store.Store
	logger   *zap.Logger
	metrics  *observability.Metrics
	cmdCh    chan CommandRequest
	subs     map[string]*Subscriber
	snapshot int64
	autoDM   *agent.AutoDM
}

func NewRoomActor(ctx context.Context, roomID string, st *store.Store, logger *zap.Logger, metrics *observability.Metrics, snapshotInterval int64, autoDM *agent.AutoDM) (*RoomActor, error) {
	ra := &RoomActor{
		RoomID:   roomID,
		store:    st,
		logger:   logger,
		metrics:  metrics,
		cmdCh:    make(chan CommandRequest, 256),
		subs:     make(map[string]*Subscriber),
		snapshot: snapshotInterval,
		autoDM:   autoDM,
	}
	if err := ra.loadState(ctx); err != nil {
		return nil, err
	}

	// Set up AutoDM dispatcher if enabled
	if autoDM != nil && autoDM.Enabled() {
		autoDM.SetDispatcher(ra, func() interface{} { return ra.GetState() })
	}

	go ra.loop(ctx)
	return ra, nil
}

func (ra *RoomActor) loadState(ctx context.Context) error {
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
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-ra.cmdCh:
			result, err := ra.handleCommand(ctx, req.Cmd)
			req.Response <- CommandResponse{Result: result, Err: err}
		}
	}
}

func (ra *RoomActor) handleCommand(ctx context.Context, cmd types.CommandEnvelope) (*types.CommandResult, error) {
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
	events, result, err := engine.HandleCommand(ra.state, cmd)
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
	var snap *store.Snapshot
	if len(storedEvents) > 0 && ra.state.LastSeq > 0 && ra.state.LastSeq%ra.snapshot == 0 {
		stateJSON, _ := engine.MarshalState(ra.state)
		snap = &store.Snapshot{RoomID: ra.RoomID, LastSeq: ra.state.LastSeq + int64(len(storedEvents)), StateJSON: stateJSON, CreatedAt: time.Now().UTC()}
	}
	if err := ra.store.AppendEvents(ctx, ra.RoomID, storedEvents, &dedupRec, snap); err != nil {
		return nil, err
	}
	for i := range storedEvents {
		storedEvents[i].Seq = ra.state.LastSeq + int64(i+1)
		payload := toEventPayload(storedEvents[i])
		ra.state.Reduce(payload)
	}
	if len(storedEvents) > 0 {
		result.AppliedSeqFrom = storedEvents[0].Seq
		result.AppliedSeqTo = storedEvents[len(storedEvents)-1].Seq
	}
	rj, _ := json.Marshal(result)
	dedupRec.ResultJSON = string(rj)
	ra.broadcast(ctx, storedEvents)
	return result, nil
}

func (ra *RoomActor) broadcast(ctx context.Context, events []store.StoredEvent) {
	ra.mu.Lock()
	defer ra.mu.Unlock()
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
			projected := projection.Project(ev, ra.state, viewer)
			if projected != nil {
				sub.Send(*projected)
			}
		}

		// Notify AutoDM to respond to game events
		if ra.autoDM != nil && ra.autoDM.Enabled() {
			go ra.autoDM.OnEvent(ctx, ev, ra.state.Copy())
		}
	}
}

func (ra *RoomActor) Subscribe(id string, s *Subscriber) {
	ra.mu.Lock()
	defer ra.mu.Unlock()
	ra.subs[id] = s
}

func (ra *RoomActor) Unsubscribe(id string) {
	ra.mu.Lock()
	defer ra.mu.Unlock()
	delete(ra.subs, id)
}

func (ra *RoomActor) Dispatch(cmd types.CommandEnvelope) CommandResponse {
	ch := make(chan CommandResponse, 1)
	ra.cmdCh <- CommandRequest{Cmd: cmd, Response: ch}
	return <-ch
}

// DispatchAsync implements the agent.CommandDispatcher interface.
// It dispatches commands asynchronously without blocking.
func (ra *RoomActor) DispatchAsync(cmd types.CommandEnvelope) error {
	resp := ra.Dispatch(cmd)
	return resp.Err
}

func (ra *RoomActor) GetState() engine.State {
	return ra.state.Copy()
}

type RoomManager struct {
	mu       sync.Mutex
	actors   map[string]*RoomActor
	store    *store.Store
	logger   *zap.Logger
	metrics  *observability.Metrics
	snapshot int64
	autoDM   *agent.AutoDM
}

func NewRoomManager(st *store.Store, logger *zap.Logger, metrics *observability.Metrics, snapshotInterval int64, autoDM *agent.AutoDM) *RoomManager {
	return &RoomManager{
		actors:   make(map[string]*RoomActor),
		store:    st,
		logger:   logger,
		metrics:  metrics,
		snapshot: snapshotInterval,
		autoDM:   autoDM,
	}
}

func (m *RoomManager) GetOrCreate(ctx context.Context, roomID string) (*RoomActor, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ra, ok := m.actors[roomID]; ok {
		return ra, nil
	}
	ra, err := NewRoomActor(ctx, roomID, m.store, m.logger, m.metrics, m.snapshot, m.autoDM)
	if err != nil {
		return nil, err
	}
	m.actors[roomID] = ra
	return ra, nil
}
