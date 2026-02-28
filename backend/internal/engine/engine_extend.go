// Package engine 白天讨论延长时间命令
//
// 玩家在白天讨论阶段可发送 extend_time 命令延长讨论时间，
// 每天最多 MaxExtensions 次，每次延长 ExtensionDurationSec 秒。
package engine

import (
	"fmt"
	"time"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

// handleExtendTime processes the extend_time command.
// Only valid during PhaseDay + SubPhaseDiscussion, within MaxExtensions limit.
func handleExtendTime(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	if state.Phase != PhaseDay || state.SubPhase != SubPhaseDiscussion {
		return nil, nil, fmt.Errorf("engine.handleExtendTime: can only extend during day discussion")
	}
	if state.ExtensionsUsed >= state.Config.MaxExtensions {
		return nil, nil, fmt.Errorf("engine.handleExtendTime: max extensions reached (%d/%d)", state.ExtensionsUsed, state.Config.MaxExtensions)
	}

	extensionDur := time.Duration(state.Config.ExtensionDurationSec) * time.Second
	newDeadline := time.Now().Add(extensionDur).UnixMilli()
	remaining := state.Config.MaxExtensions - state.ExtensionsUsed - 1

	event := newEvent(cmd, "time.extended", map[string]string{
		"deadline":             fmt.Sprintf("%d", newDeadline),
		"extensions_remaining": fmt.Sprintf("%d", remaining),
	})

	return []types.Event{event}, acceptedResult(cmd.CommandID), nil
}
