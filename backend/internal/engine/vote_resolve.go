// Package engine 投票结算统一入口
//
// resolveVoteAndCheckWin 为 handleVote（全票自动结算）和
// handleCloseVote（autodm 强制结算）提供唯一结算路径，保证：
//   - 阈值计算一致：(aliveCount+1)/2
//   - 事件字段一致：nomination.resolved(votes_for, votes_against, threshold)
//   - 处决后用 state.Copy() 再做终局检查
package engine

import (
	"fmt"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

// resolveVoteAndCheckWin tallies votes, resolves the nomination, and checks
// for win conditions in one atomic step. Returns the resolution result string
// ("executed" / "not_executed") and the combined events slice.
func resolveVoteAndCheckWin(state State, cmd types.CommandEnvelope) (string, []types.Event) {
	result, events := resolveNomination(state, cmd)

	if result != "executed" {
		return result, events
	}

	// Apply death to a copy before checking win condition,
	// because the emitted player.died hasn't been reduced yet.
	stateCopy := state.Copy()
	nomineeID := state.Nomination.Nominee
	if p, ok := stateCopy.Players[nomineeID]; ok {
		p.Alive = false
		stateCopy.Players[nomineeID] = p
	}
	stateCopy.ExecutedToday = nomineeID

	winEvents := checkWinCondition(stateCopy, cmd)
	events = append(events, winEvents...)

	return result, events
}

// resolveNomination tallies the current nomination's votes and produces
// nomination.resolved + execution events. It does NOT check win conditions.
func resolveNomination(state State, cmd types.CommandEnvelope) (string, []types.Event) {
	nom := state.Nomination

	yesVotes := 0
	for _, v := range nom.Votes {
		if v {
			yesVotes++
		}
	}

	aliveCount := state.GetAliveCount()
	// Official BotC rule: votes >= ceil(alive/2) i.e. >=50% rounded up.
	// Go integer arithmetic: (n+1)/2 == ceil(n/2).
	threshold := (aliveCount + 1) / 2

	result := "not_executed"
	if yesVotes >= threshold {
		result = "executed"
	}

	events := []types.Event{
		newEvent(cmd, "nomination.resolved", map[string]string{
			"result":        result,
			"votes_for":     fmt.Sprintf("%d", yesVotes),
			"votes_against": fmt.Sprintf("%d", len(nom.Votes)-yesVotes),
			"threshold":     fmt.Sprintf("%d", threshold),
		}),
	}

	if result == "executed" {
		events = append(events, newEvent(cmd, "execution.resolved", map[string]string{
			"result":   "executed",
			"executed": nom.Nominee,
		}))
		events = append(events, newEvent(cmd, "player.died", map[string]string{
			"user_id": nom.Nominee,
			"cause":   "execution",
		}))
	}

	return result, events
}
