package engine

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/game"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

var (
	ErrPhaseEnded       = errors.New("game already ended")
	ErrInvalidPhase     = errors.New("invalid phase for this action")
	ErrPlayerNotFound   = errors.New("player not found")
	ErrInvalidTarget    = errors.New("invalid target")
	ErrAlreadyNominated = errors.New("already nominated today")
	ErrAlreadyVoted     = errors.New("already voted")
	ErrNoGhostVote      = errors.New("no ghost vote remaining")
	ErrNominationActive = errors.New("nomination already in progress")
)

func HandleCommand(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	if state.Phase == PhaseEnded {
		return nil, nil, ErrPhaseEnded
	}
	switch cmd.Type {
	case "join":
		return handleJoin(state, cmd)
	case "leave":
		return handleLeave(state, cmd)
	case "claim_seat":
		return handleClaimSeat(state, cmd)
	case "start_game":
		return handleStartGame(state, cmd)
	case "public_chat":
		return handlePublicChat(state, cmd)
	case "whisper":
		return handleWhisper(state, cmd)
	case "nominate":
		return handleNomination(state, cmd)
	case "end_defense":
		return handleEndDefense(state, cmd)
	case "vote":
		return handleVote(state, cmd)
	case "resolve_nomination":
		return handleResolveNomination(state, cmd)
	case "ability.use":
		return handleAbility(state, cmd)
	case "advance_phase":
		return handleAdvancePhase(state, cmd)
	case "write_event":
		return handleWriteEvent(state, cmd)
	case "slayer_shot":
		return handleSlayerShot(state, cmd)
	default:
		return nil, nil, fmt.Errorf("unknown command type: %s", cmd.Type)
	}
}

func handleJoin(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	if _, exists := state.Players[cmd.ActorUserID]; exists {
		return nil, nil, fmt.Errorf("player already joined")
	}
	if state.Phase != PhaseLobby {
		return nil, nil, fmt.Errorf("cannot join after game started")
	}

	var payload map[string]string
	_ = json.Unmarshal(cmd.Payload, &payload)

	name := payload["name"]
	if name == "" {
		name = fmt.Sprintf("玩家%d", len(state.Players)+1)
	}

	eventPayload := map[string]string{
		"role":        "player",
		"name":        name,
		"seat_number": fmt.Sprintf("%d", len(state.Players)+1),
	}

	return []types.Event{newEvent(cmd, "player.joined", eventPayload)}, acceptedResult(cmd.CommandID), nil
}

func handleLeave(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	if _, exists := state.Players[cmd.ActorUserID]; !exists {
		return nil, nil, fmt.Errorf("player not in room")
	}
	if state.Phase != PhaseLobby {
		return nil, nil, fmt.Errorf("cannot leave after game started")
	}
	return []types.Event{newEvent(cmd, "player.left", nil)}, acceptedResult(cmd.CommandID), nil
}

func handleClaimSeat(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	if state.Phase != PhaseLobby {
		return nil, nil, fmt.Errorf("cannot claim seat after game started")
	}

	var payload map[string]string
	_ = json.Unmarshal(cmd.Payload, &payload)

	seatNum := payload["seat_number"]
	if seatNum == "" {
		return nil, nil, fmt.Errorf("seat_number required")
	}

	return []types.Event{newEvent(cmd, "seat.claimed", map[string]string{"seat_number": seatNum})}, acceptedResult(cmd.CommandID), nil
}

func handleStartGame(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	if state.Phase != PhaseLobby {
		return nil, nil, fmt.Errorf("cannot start game outside lobby")
	}

	// Count non-DM players
	playerCount := 0
	var userIDs []string
	var seatOrder []int

	for uid, p := range state.Players {
		if !p.IsDM {
			playerCount++
			userIDs = append(userIDs, uid)
			seatOrder = append(seatOrder, p.SeatNumber)
		}
	}

	if playerCount < 5 {
		return nil, nil, fmt.Errorf("need at least 5 players, have %d", playerCount)
	}
	if playerCount > 15 {
		return nil, nil, fmt.Errorf("too many players, max 15, have %d", playerCount)
	}

	// Use SetupAgent to assign roles
	setupConfig := game.SetupConfig{
		PlayerCount: playerCount,
	}
	setupAgent := game.NewSetupAgent(setupConfig)
	result, err := setupAgent.GenerateAssignments(userIDs, seatOrder)
	if err != nil {
		return nil, nil, fmt.Errorf("role assignment failed: %w", err)
	}

	events := []types.Event{newEvent(cmd, "game.started", nil)}

	// Create role assignment events
	for userID, assignment := range result.Assignments {
		role := game.GetRoleByID(assignment.Role)
		teamStr := "good"
		if role != nil && role.Team == game.TeamEvil {
			teamStr = "evil"
		}

		payload := map[string]string{
			"user_id":   userID,
			"role":      assignment.PerceivedRole,
			"true_role": assignment.TrueRole,
			"team":      teamStr,
		}

		if assignment.TrueRole == "imp" {
			payload["is_demon"] = "true"
		}
		if role != nil && role.Type == game.RoleMinion {
			payload["is_minion"] = "true"
		}

		events = append(events, newEvent(cmd, "role.assigned", payload))
	}

	// Assign bluffs to demon
	if len(result.BluffRoles) > 0 {
		bluffsJSON, _ := json.Marshal(result.BluffRoles)
		events = append(events, newEvent(cmd, "bluffs.assigned", map[string]string{
			"bluffs": string(bluffsJSON),
		}))
	}

	// Queue first night actions
	for _, action := range result.NightOrder {
		events = append(events, newEvent(cmd, "night.action.queued", map[string]string{
			"user_id": action.UserID,
			"role_id": action.RoleID,
			"order":   fmt.Sprintf("%d", action.Order),
		}))
	}

	// Transition to first night
	events = append(events, newEvent(cmd, "phase.first_night", map[string]string{}))

	return events, acceptedResult(cmd.CommandID), nil
}

func handlePublicChat(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	var payload map[string]string
	_ = json.Unmarshal(cmd.Payload, &payload)
	if payload == nil {
		payload = map[string]string{}
	}
	if payload["message"] == "" {
		return nil, nil, fmt.Errorf("message required")
	}

	player := state.Players[cmd.ActorUserID]
	payload["sender_name"] = player.Name
	if payload["sender_name"] == "" {
		// Fallback to ActorUserID if player not found in state (e.g. guest or DM)
		payload["sender_name"] = cmd.ActorUserID
		// Try to fix guest seat number
		if player.SeatNumber == 0 {
			// Find max seat? Or just leave as 0 (spectator)
			payload["sender_seat"] = "0"
		}
	} else {
		payload["sender_seat"] = fmt.Sprintf("%d", player.SeatNumber)
	}

	return []types.Event{newEvent(cmd, "public.chat", payload)}, acceptedResult(cmd.CommandID), nil
}

func handleWhisper(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	var payload map[string]string
	_ = json.Unmarshal(cmd.Payload, &payload)
	if payload == nil || payload["to_user_id"] == "" || payload["message"] == "" {
		return nil, nil, fmt.Errorf("invalid whisper payload")
	}
	if _, ok := state.Players[payload["to_user_id"]]; !ok {
		return nil, nil, fmt.Errorf("recipient not in room")
	}

	sender := state.Players[cmd.ActorUserID]
	payload["sender_name"] = sender.Name
	payload["sender_seat"] = fmt.Sprintf("%d", sender.SeatNumber)

	return []types.Event{newEvent(cmd, "whisper.sent", payload)}, acceptedResult(cmd.CommandID), nil
}

func handleNomination(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	if state.Phase != PhaseDay && state.Phase != PhaseNomination {
		return nil, nil, ErrInvalidPhase
	}
	if state.Nomination != nil && !state.Nomination.Resolved {
		return nil, nil, ErrNominationActive
	}

	nominator := state.Players[cmd.ActorUserID]
	if !nominator.Alive {
		return nil, nil, fmt.Errorf("dead players cannot nominate")
	}
	if nominator.HasNominated {
		return nil, nil, ErrAlreadyNominated
	}

	var payload map[string]string
	_ = json.Unmarshal(cmd.Payload, &payload)
	nomineeID := payload["nominee"]
	if nomineeID == "" {
		return nil, nil, fmt.Errorf("nominee required")
	}

	nominee, ok := state.Players[nomineeID]
	if !ok {
		return nil, nil, ErrPlayerNotFound
	}
	if nominee.WasNominated {
		return nil, nil, fmt.Errorf("player already nominated today")
	}

	events := []types.Event{
		newEvent(cmd, "nomination.created", map[string]string{
			"nominee":        nomineeID,
			"nominee_seat":   fmt.Sprintf("%d", nominee.SeatNumber),
			"nominator_seat": fmt.Sprintf("%d", nominator.SeatNumber),
		}),
	}

	// Check for Virgin ability
	if nominee.TrueRole == "virgin" && !nominee.IsPoisoned {
		if nominator.Team == "good" && game.GetRoleByID(nominator.TrueRole).Type == game.RoleTownsfolk {
			// Townsfolk nominated virgin - nominator dies
			events = append(events, newEvent(cmd, "player.died", map[string]string{
				"user_id": cmd.ActorUserID,
				"cause":   "virgin_ability",
			}))
			events = append(events, newEvent(cmd, "nomination.resolved", map[string]string{
				"result": "cancelled",
				"reason": "virgin_triggered",
			}))
		}
	}

	return events, acceptedResult(cmd.CommandID), nil
}

func handleEndDefense(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	if state.Nomination == nil || state.SubPhase != SubPhaseDefense {
		return nil, nil, fmt.Errorf("no defense phase active")
	}

	// Only nominator, nominee, or DM can end defense
	isNominator := cmd.ActorUserID == state.Nomination.Nominator
	isNominee := cmd.ActorUserID == state.Nomination.Nominee
	isDM := state.Players[cmd.ActorUserID].IsDM

	if !isNominator && !isNominee && !isDM {
		return nil, nil, fmt.Errorf("only nominator, nominee, or DM can end defense")
	}

	return []types.Event{newEvent(cmd, "defense.ended", nil)}, acceptedResult(cmd.CommandID), nil
}

func handleVote(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	if state.Nomination == nil || state.Nomination.Resolved {
		return nil, nil, fmt.Errorf("no active nomination")
	}
	if state.SubPhase != SubPhaseVoting {
		return nil, nil, fmt.Errorf("not in voting phase")
	}

	voter := state.Players[cmd.ActorUserID]

	// Check if already voted
	if _, voted := state.Nomination.Votes[cmd.ActorUserID]; voted {
		return nil, nil, ErrAlreadyVoted
	}

	// Dead players can only vote if they have ghost vote
	if !voter.Alive && !voter.HasGhostVote {
		return nil, nil, ErrNoGhostVote
	}

	// Butler check
	if voter.TrueRole == "butler" && voter.ButlerMaster != "" {
		if _, masterVoted := state.Nomination.Votes[voter.ButlerMaster]; !masterVoted {
			return nil, nil, fmt.Errorf("butler must wait for master to vote first")
		}
	}

	var payload map[string]string
	_ = json.Unmarshal(cmd.Payload, &payload)
	vote := payload["vote"]
	if vote != "yes" && vote != "no" {
		return nil, nil, fmt.Errorf("vote must be yes or no")
	}

	events := []types.Event{newEvent(cmd, "vote.cast", map[string]string{
		"vote":       vote,
		"voter_seat": fmt.Sprintf("%d", voter.SeatNumber),
	})}

	// Check if all alive players have voted
	allVoted := true
	state.Nomination.Votes[cmd.ActorUserID] = vote == "yes"
	for uid, p := range state.Players {
		if p.Alive || p.HasGhostVote {
			if _, voted := state.Nomination.Votes[uid]; !voted {
				allVoted = false
				break
			}
		}
	}

	if allVoted {
		// Auto-resolve nomination
		result, resolveEvents := resolveNomination(state, cmd)
		events = append(events, resolveEvents...)

		// Check win condition after potential execution
		if result == "executed" {
			winEvents := checkWinCondition(state, cmd)
			events = append(events, winEvents...)
		}
	}

	return events, acceptedResult(cmd.CommandID), nil
}

func handleResolveNomination(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	if state.Nomination == nil {
		return nil, nil, fmt.Errorf("no active nomination")
	}

	_, events := resolveNomination(state, cmd)
	return events, acceptedResult(cmd.CommandID), nil
}

func resolveNomination(state State, cmd types.CommandEnvelope) (string, []types.Event) {
	nom := state.Nomination

	yesVotes := 0
	for _, v := range nom.Votes {
		if v {
			yesVotes++
		}
	}

	aliveCount := state.GetAliveCount()
	threshold := (aliveCount / 2) + 1

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

func handleAbility(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	if state.Phase != PhaseNight && state.Phase != PhaseFirstNight {
		return nil, nil, fmt.Errorf("abilities only at night")
	}

	player := state.Players[cmd.ActorUserID]

	var payload map[string]string
	_ = json.Unmarshal(cmd.Payload, &payload)

	// Build game context for night agent
	ctx := buildGameContext(state)
	nightAgent := game.NewNightAgent(ctx)

	var targetIDs []string
	if targets := payload["targets"]; targets != "" {
		_ = json.Unmarshal([]byte(targets), &targetIDs)
	}
	if target := payload["target"]; target != "" {
		targetIDs = []string{target}
	}

	req := game.AbilityRequest{
		UserID:       cmd.ActorUserID,
		RoleID:       player.TrueRole,
		TargetIDs:    targetIDs,
		ActionType:   payload["action_type"],
		NightNumber:  state.NightCount,
		IsFirstNight: state.Phase == PhaseFirstNight,
	}

	result, err := nightAgent.ResolveAbility(req)
	if err != nil {
		return nil, nil, err
	}

	events := []types.Event{}

	// Process effects
	for _, effect := range result.Effects {
		switch effect.Type {
		case "kill":
			events = append(events, newEvent(cmd, "player.died", map[string]string{
				"user_id": effect.TargetID,
				"cause":   "demon",
			}))
		case "protect":
			events = append(events, newEvent(cmd, "player.protected", map[string]string{
				"user_id": effect.TargetID,
			}))
		case "poison":
			events = append(events, newEvent(cmd, "player.poisoned", map[string]string{
				"user_id": effect.TargetID,
			}))
		case "starpass":
			// The old demon dies
			events = append(events, newEvent(cmd, "player.died", map[string]string{
				"user_id": effect.TargetID,
				"cause":   "starpass",
			}))
			// Find a minion to become demon
			var candidateMinions []string
			var scarletWomanID string

			for _, minionID := range state.MinionIDs {
				p := state.Players[minionID]
				if p.Alive {
					candidateMinions = append(candidateMinions, minionID)
					if p.TrueRole == "scarlet_woman" {
						scarletWomanID = minionID
					}
				}
			}

			if len(candidateMinions) > 0 {
				newDemonID := ""
				// Priority: Scarlet Woman -> Random Minion
				if scarletWomanID != "" {
					newDemonID = scarletWomanID
				} else {
					// Randomly select a minion
					// Use crypto/rand for secure random selection
					idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(candidateMinions))))
					if err != nil {
						// Fallback to first if random fails
						newDemonID = candidateMinions[0]
					} else {
						newDemonID = candidateMinions[idx.Int64()]
					}
				}

				events = append(events, newEvent(cmd, "demon.changed", map[string]string{
					"old_demon": cmd.ActorUserID,
					"new_demon": newDemonID,
				}))
			}
		}
	}

	targetsJSON, _ := json.Marshal(targetIDs)
	events = append(events, newEvent(cmd, "night.action.completed", map[string]string{
		"user_id": cmd.ActorUserID,
		"role_id": player.TrueRole,
		"targets": string(targetsJSON),
		"result":  result.Message,
	}))

	return events, acceptedResult(cmd.CommandID), nil
}

func handleAdvancePhase(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	var payload map[string]string
	_ = json.Unmarshal(cmd.Payload, &payload)

	targetPhase := payload["phase"]
	events := []types.Event{}

	switch targetPhase {
	case "day":
		// Announce deaths from night
		for _, death := range state.PendingDeaths {
			if !death.Protected {
				events = append(events, newEvent(cmd, "player.died", map[string]string{
					"user_id": death.UserID,
					"cause":   death.Cause,
				}))
				// Update local state for immediate win check
				if p, ok := state.Players[death.UserID]; ok {
					p.Alive = false
					state.Players[death.UserID] = p
				}
			}
		}
		// Clear poison
		events = append(events, newEvent(cmd, "poison.cleared", nil))
		events = append(events, newEvent(cmd, "phase.day", nil))

	case "night":
		events = append(events, newEvent(cmd, "phase.night", nil))

	case "nomination":
		events = append(events, newEvent(cmd, "phase.nomination", nil))

	default:
		return nil, nil, fmt.Errorf("invalid target phase: %s", targetPhase)
	}

	// Check win condition
	winEvents := checkWinCondition(state, cmd)
	events = append(events, winEvents...)

	return events, acceptedResult(cmd.CommandID), nil
}

func handleWriteEvent(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	if cmd.ActorUserID != "autodm" && cmd.ActorUserID != "auto-dm" {
		player, ok := state.Players[cmd.ActorUserID]
		if !ok || !player.IsDM {
			return nil, nil, fmt.Errorf("only DM or AutoDM can write custom events")
		}
	}

	var payload struct {
		EventType string                 `json:"event_type"`
		Data      map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(cmd.Payload, &payload); err != nil {
		return nil, nil, fmt.Errorf("invalid write_event payload: %w", err)
	}
	if payload.EventType == "" {
		return nil, nil, fmt.Errorf("event_type required")
	}

	data := make(map[string]string, len(payload.Data))
	for k, v := range payload.Data {
		switch vv := v.(type) {
		case string:
			data[k] = vv
		default:
			b, err := json.Marshal(v)
			if err != nil {
				data[k] = fmt.Sprint(v)
				continue
			}
			data[k] = string(b)
		}
	}

	return []types.Event{newEvent(cmd, payload.EventType, data)}, acceptedResult(cmd.CommandID), nil
}

func handleSlayerShot(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	if state.Phase != PhaseDay {
		return nil, nil, fmt.Errorf("slayer can only shoot during day")
	}

	slayer := state.Players[cmd.ActorUserID]
	if slayer.TrueRole != "slayer" {
		return nil, nil, fmt.Errorf("only slayer can use this ability")
	}

	// Check if slayer has already used ability
	for _, r := range slayer.Reminders {
		if r == "无能力" {
			return nil, nil, fmt.Errorf("slayer has already used ability")
		}
	}

	var payload map[string]string
	_ = json.Unmarshal(cmd.Payload, &payload)
	targetID := payload["target"]
	if targetID == "" {
		return nil, nil, fmt.Errorf("target required")
	}

	target, ok := state.Players[targetID]
	if !ok {
		return nil, nil, ErrPlayerNotFound
	}

	events := []types.Event{
		newEvent(cmd, "slayer.shot", map[string]string{
			"target":      targetID,
			"target_seat": fmt.Sprintf("%d", target.SeatNumber),
		}),
	}

	// Mark slayer as having used ability
	// This would be handled by adding reminder

	// If target is demon (and slayer not poisoned), they die
	if target.TrueRole == "imp" && !slayer.IsPoisoned {
		events = append(events, newEvent(cmd, "player.died", map[string]string{
			"user_id": targetID,
			"cause":   "slayer",
		}))

		// Update state for win check (local copy only, doesn't affect persistence)
		target.Alive = false
		state.Players[targetID] = target

		// Check win condition
		winEvents := checkWinCondition(state, cmd)
		events = append(events, winEvents...)
	}

	return events, acceptedResult(cmd.CommandID), nil
}

func checkWinCondition(state State, cmd types.CommandEnvelope) []types.Event {
	// Create a copy and apply pending changes
	stateCopy := state.Copy()

	ended, winner, reason := stateCopy.CheckWinCondition()
	if ended {
		return []types.Event{newEvent(cmd, "game.ended", map[string]string{
			"winner": winner,
			"reason": reason,
		})}
	}

	// Check if demon died but game continues (Scarlet Woman case)
	if demon, ok := stateCopy.Players[stateCopy.DemonID]; ok && !demon.Alive {
		// Find a living Scarlet Woman
		for uid, p := range stateCopy.Players {
			if p.TrueRole == "scarlet_woman" && p.Alive {
				// We also check alive count >= 5 to be safe, though CheckWinCondition
				// already implicitly checked this by returning ended=false.
				if stateCopy.GetAliveCount() >= 5 {
					return []types.Event{
						newEvent(cmd, "demon.changed", map[string]string{
							"old_demon": stateCopy.DemonID,
							"new_demon": uid,
							"reason":    "scarlet_woman",
						}),
						// Notify the new demon privately
						newEvent(cmd, "role.change", map[string]string{
							"user_id":  uid,
							"new_role": "imp", // She becomes the Imp
							"reason":   "scarlet_woman_inheritance",
						}),
					}
				}
			}
		}
	}

	return nil
}

func buildGameContext(state State) *game.GameContext {
	ctx := &game.GameContext{
		Players:      make(map[string]*game.PlayerState),
		SeatOrder:    state.SeatOrder,
		PoisonedIDs:  make(map[string]bool),
		ProtectedIDs: make(map[string]bool),
		DeadIDs:      make(map[string]bool),
		DemonID:      state.DemonID,
		MinionIDs:    state.MinionIDs,
		NightNumber:  state.NightCount,
	}

	for uid, p := range state.Players {
		team := game.TeamGood
		if p.Team == "evil" {
			team = game.TeamEvil
		}
		ctx.Players[uid] = &game.PlayerState{
			UserID:      uid,
			SeatNumber:  p.SeatNumber,
			Role:        p.Role,
			TrueRole:    p.TrueRole,
			Team:        team,
			IsAlive:     p.Alive,
			IsPoisoned:  p.IsPoisoned,
			IsProtected: p.IsProtected,
		}

		if p.IsPoisoned {
			ctx.PoisonedIDs[uid] = true
		}
		if p.IsProtected {
			ctx.ProtectedIDs[uid] = true
		}
		if !p.Alive {
			ctx.DeadIDs[uid] = true
		}
		if p.TrueRole == "drunk" {
			ctx.DrunkID = uid
		}
	}

	// Set red herring for fortune teller
	for uid, p := range state.Players {
		if p.TrueRole == "fortuneteller" {
			// Pick a random good player as red herring
			for targetUID, target := range state.Players {
				if target.Team == "good" && targetUID != uid && target.Alive {
					ctx.RedHerringID = targetUID
					break
				}
			}
			break
		}
	}

	return ctx
}

func newEvent(cmd types.CommandEnvelope, eventType string, payload map[string]string) types.Event {
	b, _ := json.Marshal(payload)
	return types.Event{
		RoomID:            cmd.RoomID,
		Seq:               0,
		EventID:           uuid.NewString(),
		EventType:         eventType,
		ActorUserID:       cmd.ActorUserID,
		CausationCommand:  cmd.CommandID,
		Payload:           b,
		ServerTimestampMs: time.Now().UnixMilli(),
	}
}

func acceptedResult(commandID string) *types.CommandResult {
	return &types.CommandResult{CommandID: commandID, Status: "accepted"}
}

func mustMarshal(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
