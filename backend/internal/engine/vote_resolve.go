// Package engine 投票结算统一入口
//
// resolveVoteAndCheckWin 为 handleVote（全票自动结算）和
// handleCloseVote（autodm 强制结算）提供唯一结算路径，保证：
//   - 阈值计算一致：(aliveCount+1)/2
//   - 事件字段一致：nomination.resolved(votes_for, votes_against, threshold)
//   - "待处决"(on_the_block) 延迟处决：投票达标不立即处决，
//     而是记录到 OnTheBlock，白天结束时统一处决得票最高者
package engine

import (
	"fmt"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

// resolveVoteAndCheckWin tallies votes, resolves the nomination using on-the-block
// rules, and returns the resolution result string and combined events slice.
// Execution is deferred to handleAdvancePhase("night").
func resolveVoteAndCheckWin(state State, cmd types.CommandEnvelope) (string, []types.Event) {
	result, events := resolveNomination(state, cmd)
	return result, events
}

// resolveNomination tallies the current nomination's votes and produces
// nomination.resolved events. Uses "on_the_block" pattern:
//   - votes >= threshold → "on_the_block" (replaces current if strictly more votes)
//   - votes >= threshold but == current on-block votes → "tied" (clears block)
//   - votes < threshold → "not_on_the_block"
func resolveNomination(state State, cmd types.CommandEnvelope) (string, []types.Event) {
	nom := state.Nomination

	yesVotes := 0
	for _, v := range nom.Votes {
		if v {
			yesVotes++
		}
	}

	aliveCount := state.GetAliveCount()
	threshold := (aliveCount + 1) / 2

	result := determineBlockResult(yesVotes, threshold, state.OnTheBlock)

	events := []types.Event{
		newEvent(cmd, "nomination.resolved", map[string]string{
			"result":        result,
			"votes_for":     fmt.Sprintf("%d", yesVotes),
			"votes_against": fmt.Sprintf("%d", len(nom.Votes)-yesVotes),
			"threshold":     fmt.Sprintf("%d", threshold),
		}),
	}

	return result, events
}

// determineBlockResult decides the nomination outcome per official BotC rules.
func determineBlockResult(yesVotes, threshold int, current *OnTheBlockInfo) string {
	if yesVotes < threshold {
		return "not_on_the_block"
	}
	if current == nil {
		return "on_the_block"
	}
	if yesVotes > current.VotesFor {
		return "on_the_block"
	}
	if yesVotes == current.VotesFor {
		return "tied"
	}
	return "not_on_the_block"
}
