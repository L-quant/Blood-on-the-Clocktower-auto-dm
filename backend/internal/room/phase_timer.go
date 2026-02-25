// Package room 阶段超时计时器
//
// PhaseTimer 在指定时长后以 autodm 身份向 RoomActor 发送命令，
// 用于辩护超时、投票超时、夜晚行动超时等场景。
// 每次 Schedule 自动取消上一个计时器，防止陈旧超时误触发。
package room

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

// PhaseTimer schedules a single timeout command. Scheduling a new timeout
// automatically cancels any pending one (at most one timer active).
type PhaseTimer struct {
	mu       sync.Mutex
	timer    *time.Timer
	roomID   string
	dispatch func(types.CommandEnvelope)
	logger   *zap.Logger
}

// NewPhaseTimer creates a timer bound to a room.
// dispatch is called on the RoomActor's goroutine to inject the command.
func NewPhaseTimer(roomID string, dispatch func(types.CommandEnvelope), logger *zap.Logger) *PhaseTimer {
	return &PhaseTimer{
		roomID:   roomID,
		dispatch: dispatch,
		logger:   logger,
	}
}

// Schedule sets a timeout that fires cmd after dur. Any pending timer is cancelled.
func (pt *PhaseTimer) Schedule(dur time.Duration, cmdType string, data map[string]string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if pt.timer != nil {
		pt.timer.Stop()
		pt.timer = nil
	}

	pt.timer = time.AfterFunc(dur, func() {
		payload, _ := json.Marshal(data)
		cmd := types.CommandEnvelope{
			CommandID:   uuid.NewString(),
			RoomID:      pt.roomID,
			Type:        cmdType,
			ActorUserID: "autodm",
			Payload:     payload,
		}
		pt.logger.Debug("phase timer fired",
			zap.String("room_id", pt.roomID),
			zap.String("command", cmdType),
		)
		pt.dispatch(cmd)
	})
}

// Cancel stops any pending timer.
func (pt *PhaseTimer) Cancel() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if pt.timer != nil {
		pt.timer.Stop()
		pt.timer = nil
	}
}
