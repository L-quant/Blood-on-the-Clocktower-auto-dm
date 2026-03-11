// Package engine 游戏命令处理器，路由 28 种命令到具体 handler 并生成事件
//
// [IN]  internal/game（角色定义、夜晚行动解析、游戏初始化）
// [IN]  internal/types（命令与事件类型）
// [OUT] room（HandleCommand 命令分发）
// [OUT] agent（状态类型与工具调用）
// [POS] 游戏状态机核心，所有游戏逻辑的中枢
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
	case "room_settings":
		return handleRoomSettings(state, cmd)
	case "start_game":
		return handleStartGame(state, cmd)
	case "public_chat":
		return handlePublicChat(state, cmd)
	case "whisper":
		return handleWhisper(state, cmd)
	case "evil_team_chat":
		return handleEvilTeamChat(state, cmd)
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
	// FIX-12/13/14: Handle autodm-only command types
	case "close_vote":
		return handleCloseVote(state, cmd)
	case "request_action":
		return handleRequestAction(state, cmd)
	case "set_timer":
		return handleSetTimer(state, cmd)
	case "extend_time":
		return handleExtendTime(state, cmd)
	case "night_timeout":
		return handleNightTimeout(state, cmd)
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

func handleRoomSettings(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	if state.Phase != PhaseLobby {
		return nil, nil, fmt.Errorf("cannot change settings after game started")
	}

	var payload map[string]string
	_ = json.Unmarshal(cmd.Payload, &payload)

	eventPayload := map[string]string{}
	if ed, ok := payload["edition"]; ok {
		eventPayload["edition"] = ed
	}
	if mp, ok := payload["max_players"]; ok {
		eventPayload["max_players"] = mp
	}

	return []types.Event{newEvent(cmd, "room.settings.changed", eventPayload)}, acceptedResult(cmd.CommandID), nil
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

	// Parse optional custom_roles from payload (injected by AI Composer)
	var payload map[string]string
	_ = json.Unmarshal(cmd.Payload, &payload)
	var customRoles []string
	if cr, ok := payload["custom_roles"]; ok && cr != "" {
		_ = json.Unmarshal([]byte(cr), &customRoles)
	}

	// Use SetupAgent to assign roles
	setupConfig := game.SetupConfig{
		PlayerCount: playerCount,
		Edition:     state.Edition,
		CustomRoles: customRoles,
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

		// Spy: emit apparent role for info resolution
		if assignment.SpyApparentRole != "" {
			payload["spy_apparent_role"] = assignment.SpyApparentRole
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

	// Assign red herring for fortune teller (a good player who isn't the fortune teller)
	var fortuneTellerID string
	var goodPlayerIDs []string
	for userID, assignment := range result.Assignments {
		if assignment.TrueRole == "fortuneteller" {
			fortuneTellerID = userID
		}
		if assignment.Team == game.TeamGood && assignment.TrueRole != "fortuneteller" {
			goodPlayerIDs = append(goodPlayerIDs, userID)
		}
	}
	if fortuneTellerID != "" && len(goodPlayerIDs) > 0 {
		rhIdx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(goodPlayerIDs))))
		events = append(events, newEvent(cmd, "red_herring.assigned", map[string]string{
			"user_id": goodPlayerIDs[rhIdx.Int64()],
		}))
	}

	// Queue first night actions
	for _, action := range result.NightOrder {
		actionType := ""
		if r := game.GetRoleByID(action.RoleID); r != nil {
			actionType = string(r.FirstNightActionType)
		}
		events = append(events, newEvent(cmd, "night.action.queued", map[string]string{
			"user_id":     action.UserID,
			"role_id":     action.RoleID,
			"order":       fmt.Sprintf("%d", action.Order),
			"action_type": actionType,
		}))
	}
	// Auto-complete no_action roles (e.g. Imp first night)
	events = append(events, buildNoActionCompletions(cmd, result.NightOrder)...)

	// Transition to first night
	events = append(events, newEvent(cmd, "phase.first_night", map[string]string{}))

	// 首夜开始时：邪恶阵营互认（爪牙认恶魔、恶魔认爪牙+伪装角色）
	events = append(events, buildTeamRecognitionFromSetup(cmd, result)...)

	// Prompt the first actionable player (sequential night actions)
	// Build NightAction slice matching engine state format for prompt helper
	queuedActions := buildEngineNightActions(result.NightOrder, true)
	autoCompleted := buildNoActionSet(result.NightOrder)
	for i := range queuedActions {
		if autoCompleted[queuedActions[i].UserID] {
			queuedActions[i].Completed = true
		}
	}
	events = append(events, buildFirstPrompt(cmd, queuedActions)...)

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
	if player.Name != "" {
		payload["sender_name"] = player.Name
		payload["sender_seat"] = fmt.Sprintf("%d", player.SeatNumber)
	} else {
		payload["sender_name"] = cmd.ActorUserID
		payload["sender_seat"] = "0"
	}

	return []types.Event{newEvent(cmd, "public.chat", payload)}, acceptedResult(cmd.CommandID), nil
}

func handleEvilTeamChat(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	player, ok := state.Players[cmd.ActorUserID]
	if !ok {
		return nil, nil, fmt.Errorf("player not found")
	}
	if player.Team != "evil" {
		return nil, nil, fmt.Errorf("only evil players can use evil team chat")
	}

	var payload map[string]string
	_ = json.Unmarshal(cmd.Payload, &payload)
	if payload == nil || payload["message"] == "" {
		return nil, nil, fmt.Errorf("message required")
	}

	payload["sender_name"] = player.Name
	payload["sender_seat"] = fmt.Sprintf("%d", player.SeatNumber)

	return []types.Event{newEvent(cmd, "evil_team.chat", payload)}, acceptedResult(cmd.CommandID), nil
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

	// FIX-17: Allow autodm to proxy nominations. When autodm sends this command,
	// the actual nominator comes from the payload "nominator" field. If absent,
	// pick the first alive player who hasn't nominated yet.
	actorID := cmd.ActorUserID
	if actorID == "autodm" {
		var payload map[string]string
		_ = json.Unmarshal(cmd.Payload, &payload)
		if nominatorID, ok := payload["nominator"]; ok && nominatorID != "" {
			actorID = nominatorID
		} else {
			// Find any alive player who hasn't nominated as proxy
			for _, uid := range state.SeatOrder {
				p := state.Players[uid]
				if p.Alive && !p.HasNominated {
					actorID = uid
					break
				}
			}
		}
	}

	nominator := state.Players[actorID]
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
			"nominee":           nomineeID,
			"nominee_seat":      fmt.Sprintf("%d", nominee.SeatNumber),
			"nominator_seat":    fmt.Sprintf("%d", nominator.SeatNumber),
			"nominator_user_id": actorID,
			"vote_order":        buildVoteOrderJSON(state, nominee.SeatNumber),
		}),
	}

	// Emit timer for defense phase countdown
	defenseDeadline := time.Now().Add(time.Duration(state.Config.DefenseDurationSec) * time.Second).UnixMilli()
	events = append(events, newEvent(cmd, "timer.set", map[string]string{
		"timer_type": "defense",
		"deadline":   fmt.Sprintf("%d", defenseDeadline),
	}))

	// Check for Virgin ability — FIX-16: only triggers once per game
	if nominee.TrueRole == "virgin" && !nominee.IsPoisoned {
		virginUsed := false
		for _, r := range nominee.Reminders {
			if r == "no_ability" {
				virginUsed = true
				break
			}
		}
		if !virginUsed && nominator.Team == "good" && game.GetRoleByID(nominator.TrueRole).Type == game.RoleTownsfolk {
			// Townsfolk nominated virgin - nominator dies (use resolved actorID, not cmd.ActorUserID)
			events = append(events, newEvent(cmd, "player.died", map[string]string{
				"user_id": actorID,
				"cause":   "virgin_ability",
			}))
			// Mark virgin ability as used
			events = append(events, newEvent(cmd, "reminder.added", map[string]string{
				"user_id":  nomineeID,
				"reminder": "no_ability",
			}))
			events = append(events, newEvent(cmd, "nomination.resolved", map[string]string{
				"result": "cancelled",
				"reason": "virgin_triggered",
			}))
		}
	}

	return events, acceptedResult(cmd.CommandID), nil
}

// buildVoteOrderJSON generates the clockwise voting sequence starting from
// the seat after the nominee. Only includes eligible voters (alive or has ghost vote).
// Returns a JSON-serialized array of seat numbers for frontend consumption.
// The backend stores user_ids in Nomination.VoteOrder (built by reducer).
func buildVoteOrderJSON(state State, nomineeSeat int) string {
	n := len(state.SeatOrder)
	if n == 0 {
		return "[]"
	}
	// Find nominee index in SeatOrder
	nomineeIdx := -1
	for i, uid := range state.SeatOrder {
		if state.Players[uid].SeatNumber == nomineeSeat {
			nomineeIdx = i
			break
		}
	}
	if nomineeIdx < 0 {
		return "[]"
	}
	// Build ordered seats starting from nominee+1, wrapping around (nominee last)
	seats := []int{}
	for offset := 1; offset <= n; offset++ {
		idx := (nomineeIdx + offset) % n
		uid := state.SeatOrder[idx]
		p := state.Players[uid]
		if p.Alive || p.HasGhostVote {
			seats = append(seats, p.SeatNumber)
		}
	}
	data, _ := json.Marshal(seats)
	return string(data)
}

func handleEndDefense(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	if state.Nomination == nil || state.SubPhase != SubPhaseDefense {
		return nil, nil, fmt.Errorf("no defense phase active")
	}

	// Only nominator, nominee, DM, or autodm can end defense
	isNominator := cmd.ActorUserID == state.Nomination.Nominator
	isNominee := cmd.ActorUserID == state.Nomination.Nominee
	isDM := state.Players[cmd.ActorUserID].IsDM
	isAutoDM := cmd.ActorUserID == "autodm" || cmd.ActorUserID == "auto-dm"

	if !isNominator && !isNominee && !isDM && !isAutoDM {
		return nil, nil, fmt.Errorf("only nominator, nominee, DM, or autodm can end defense")
	}

	// Emit timer for voting phase countdown
	votingDeadline := time.Now().Add(time.Duration(state.Config.VotingDurationSec) * time.Duration(len(state.Players)) * time.Second).UnixMilli()
	events := []types.Event{
		newEvent(cmd, "defense.ended", nil),
		newEvent(cmd, "timer.set", map[string]string{
			"timer_type": "voting",
			"deadline":   fmt.Sprintf("%d", votingDeadline),
		}),
	}

	return events, acceptedResult(cmd.CommandID), nil
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

	// Sequential voting: only the current voter may vote
	if err := validateSequentialVoter(state, cmd.ActorUserID); err != nil {
		return nil, nil, err
	}

	// Butler check: butler may only vote yes if their master voted yes
	if voter.TrueRole == "butler" && voter.ButlerMaster != "" {
		masterVote, masterVoted := state.Nomination.Votes[voter.ButlerMaster]
		if !masterVoted {
			// Master hasn't voted yet — butler can only vote no
			var p map[string]string
			_ = json.Unmarshal(cmd.Payload, &p)
			if p["vote"] == "yes" {
				return nil, nil, fmt.Errorf("butler cannot vote yes until master votes yes")
			}
		} else if !masterVote {
			var p map[string]string
			_ = json.Unmarshal(cmd.Payload, &p)
			if p["vote"] == "yes" {
				return nil, nil, fmt.Errorf("butler cannot vote yes unless master votes yes")
			}
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

	// Record vote locally for auto-resolve check
	state.Nomination.Votes[cmd.ActorUserID] = vote == "yes"
	nextIdx := state.Nomination.CurrentVoterIdx + 1

	// Check if this was the last voter
	if nextIdx >= len(state.Nomination.VoteOrder) {
		_, resolveEvents := resolveVoteAndCheckWin(state, cmd)
		events = append(events, resolveEvents...)
	}

	return events, acceptedResult(cmd.CommandID), nil
}

// validateSequentialVoter checks that the actor is the current voter in order.
func validateSequentialVoter(state State, actorID string) error {
	nom := state.Nomination
	if len(nom.VoteOrder) == 0 {
		return nil // No order set (legacy), allow any voter
	}
	if nom.CurrentVoterIdx >= len(nom.VoteOrder) {
		return fmt.Errorf("all voters have already voted")
	}
	currentVoter := nom.VoteOrder[nom.CurrentVoterIdx]
	if actorID != currentVoter {
		return fmt.Errorf("not your turn to vote, waiting for seat to vote first")
	}
	return nil
}

func handleResolveNomination(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	if state.Nomination == nil {
		return nil, nil, fmt.Errorf("no active nomination")
	}

	_, events := resolveVoteAndCheckWin(state, cmd)
	return events, acceptedResult(cmd.CommandID), nil
}

func handleAbility(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	if state.Phase != PhaseNight && state.Phase != PhaseFirstNight {
		return nil, nil, fmt.Errorf("abilities only at night")
	}

	player := state.Players[cmd.ActorUserID]

	// Strict sequential enforcement: only the current action's player may act
	if err := validateCurrentNightAction(state, cmd.ActorUserID); err != nil {
		return nil, nil, err
	}

	var payload map[string]string
	_ = json.Unmarshal(cmd.Payload, &payload)

	var targetIDs []string
	if targets := payload["targets"]; targets != "" {
		_ = json.Unmarshal([]byte(targets), &targetIDs)
	}
	if target := payload["target"]; target != "" {
		targetIDs = []string{target}
	}

	events := []types.Event{}
	targetsJSON, _ := json.Marshal(targetIDs)
	completionEvent := newEvent(cmd, "night.action.completed", map[string]string{
		"user_id": cmd.ActorUserID,
		"role_id": player.TrueRole,
		"targets": string(targetsJSON),
	})

	// 收集层：仅记录意图，不调用 ResolveAbility，不生成效果事件
	events = append(events, completionEvent)

	// Prompt next player or trigger resolution
	allDone := true
	for _, a := range state.NightActions {
		if a.UserID == cmd.ActorUserID {
			continue // this one is being completed now
		}
		if !a.Completed {
			allDone = false
			break
		}
	}
	if !allDone {
		// Emit prompt for next player in sequence
		promptEvents := buildNextPrompt(cmd, state.NightActions, cmd.ActorUserID)
		events = append(events, promptEvents...)
	}
	if allDone && len(state.NightActions) > 0 {
		workingState := state.Copy()
		applyEventsToState(&workingState, []types.Event{completionEvent})

		// 所有行动收集完毕 → 统一结算 → 信息分发 → 天亮
		resolveEvents := resolveNight(workingState, cmd)
		events = append(events, resolveEvents...)

		// 应用结算效果到 state 副本，用于信息分发
		stateCopy := workingState.Copy()
		applyResolveEffects(&stateCopy, resolveEvents)

		infoEvents := distributeNightInfo(stateCopy, cmd)
		events = append(events, infoEvents...)

		events = append(events, newEvent(cmd, "phase.day", nil))

		// 胜负检查
		winEvents := checkWinCondition(stateCopy, cmd)
		events = append(events, winEvents...)
	}

	return events, acceptedResult(cmd.CommandID), nil
}

func handleAdvancePhase(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	// Permission: only autodm, room owner, or DM may advance phase
	isAutoDM := cmd.ActorUserID == "autodm" || cmd.ActorUserID == "auto-dm"
	isOwner := cmd.ActorUserID == state.OwnerID
	isDM := false
	if p, ok := state.Players[cmd.ActorUserID]; ok {
		isDM = p.IsDM
	}
	if !isAutoDM && !isOwner && !isDM {
		return nil, nil, fmt.Errorf("only room owner, DM, or autodm can advance phase")
	}

	var payload map[string]string
	_ = json.Unmarshal(cmd.Payload, &payload)

	targetPhase := payload["phase"]
	events := []types.Event{}

	if targetPhase == "day" && (state.Phase == PhaseFirstNight || state.Phase == PhaseNight) {
		return nil, nil, fmt.Errorf("night cannot be forced to day; complete all night actions instead")
	}

	switch targetPhase {
	case "day":
		// Auto-complete any remaining night actions as timed_out
		timeoutEvents, _ := CompleteRemainingNightActions(state, cmd)
		events = append(events, timeoutEvents...)
		events = append(events, finalizeNightFromCompletions(state, cmd, timeoutEvents)...)

	case "night":
		// Execute on-the-block player before entering night (only if no execution yet)
		if state.OnTheBlock != nil && state.ExecutedToday == "" {
			events = append(events, newEvent(cmd, "execution.resolved", map[string]string{
				"result":   "executed",
				"executed": state.OnTheBlock.UserID,
			}))
			events = append(events, newEvent(cmd, "player.died", map[string]string{
				"user_id": state.OnTheBlock.UserID,
				"cause":   "execution",
			}))
			if p, ok := state.Players[state.OnTheBlock.UserID]; ok {
				p.Alive = false
				state.Players[state.OnTheBlock.UserID] = p
			}
			state.ExecutedToday = state.OnTheBlock.UserID
		}

		// Clear poison at dusk (official rule: poisoned "tonight and tomorrow day")
		events = append(events, newEvent(cmd, "poison.cleared", nil))
		events = append(events, newEvent(cmd, "phase.night", nil))

		// FIX-4: Generate night.action.queued events for nights 2+
		// Build assignments from current state for night order generation
		assignments := make(map[string]game.Assignment)
		for uid, p := range state.Players {
			if p.Alive {
				assignments[uid] = game.Assignment{
					UserID:   uid,
					TrueRole: p.TrueRole,
					Team:     game.Team(p.Team),
				}
			}
		}
		allRoles := game.GetAllRoles()
		nightActions := game.GenerateNightOrder(allRoles, assignments, false)
		for _, action := range nightActions {
			actionType := ""
			if r := game.GetRoleByID(action.RoleID); r != nil {
				actionType = string(r.NightActionType)
			}
			events = append(events, newEvent(cmd, "night.action.queued", map[string]string{
				"user_id":     action.UserID,
				"role_id":     action.RoleID,
				"order":       fmt.Sprintf("%d", action.Order),
				"action_type": actionType,
			}))
		}
		// Prompt first actionable player for nights 2+
		queuedOtherNight := buildEngineNightActions(nightActions, false)
		events = append(events, buildFirstPrompt(cmd, queuedOtherNight)...)

	case "nomination":

	default:
		return nil, nil, fmt.Errorf("invalid target phase: %s", targetPhase)
	}

	if targetPhase == "day" {
		return events, acceptedResult(cmd.CommandID), nil
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
	if !isDaytimePhase(state.Phase) {
		return nil, nil, fmt.Errorf("slayer can only shoot during day")
	}

	shooter, ok := state.Players[cmd.ActorUserID]
	if !ok {
		return nil, nil, ErrPlayerNotFound
	}
	isTrueSlayer := shooter.TrueRole == "slayer"

	if isTrueSlayer {
		for _, reminder := range shooter.Reminders {
			if reminder == "no_ability" || reminder == "无能力" {
				return nil, nil, fmt.Errorf("slayer has already used ability")
			}
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

	shotResult := "no_effect"
	postShotEvents := make([]types.Event, 0, 4)
	if isTrueSlayer {
		postShotEvents = append(postShotEvents, newEvent(cmd, "reminder.added", map[string]string{
			"user_id":  cmd.ActorUserID,
			"reminder": "no_ability",
		}))
	}

	if isTrueSlayer && targetID == state.DemonID && !shooter.IsPoisoned {
		playerDiedEvent := newEvent(cmd, "player.died", map[string]string{
			"user_id": targetID,
			"cause":   "slayer",
		})
		postShotEvents = append(postShotEvents, playerDiedEvent)

		resolvedState := state.Copy()
		applyEventsToState(&resolvedState, []types.Event{playerDiedEvent})
		winEvents := checkWinCondition(resolvedState, cmd)
		postShotEvents = append(postShotEvents, winEvents...)

		if hasEventType(winEvents, "game.ended") {
			shotResult = "killed"
		} else if hasEventType(winEvents, "demon.changed") {
			applyEventsToState(&resolvedState, winEvents)
			postShotEvents = append(postShotEvents, buildNightTransitionEvents(resolvedState, cmd)...)
			shotResult = "killed_night"
		} else {
			shotResult = "killed"
		}
	}

	events := []types.Event{newEvent(cmd, "slayer.shot", map[string]string{
		"target":       targetID,
		"target_seat":  fmt.Sprintf("%d", target.SeatNumber),
		"shooter_seat": fmt.Sprintf("%d", shooter.SeatNumber),
		"result":       shotResult,
	})}
	events = append(events, postShotEvents...)

	return events, acceptedResult(cmd.CommandID), nil
}

// handleCloseVote resolves an active nomination via the unified vote settlement path.
// Only autodm may call this (timeout-driven force close).
func handleCloseVote(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	if cmd.ActorUserID != "autodm" {
		return nil, nil, fmt.Errorf("only autodm can close votes")
	}
	if state.Nomination == nil || state.Nomination.Resolved {
		return nil, nil, fmt.Errorf("no active nomination to close")
	}

	_, events := resolveVoteAndCheckWin(state, cmd)
	return events, acceptedResult(cmd.CommandID), nil
}

// FIX-13: handleRequestAction emits an event prompting a player to act.
func handleRequestAction(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	if cmd.ActorUserID != "autodm" {
		return nil, nil, fmt.Errorf("only autodm can request actions")
	}

	var payload map[string]string
	_ = json.Unmarshal(cmd.Payload, &payload)

	userID := payload["user_id"]
	if _, ok := state.Players[userID]; !ok {
		return nil, nil, fmt.Errorf("target player not found: %s", userID)
	}

	events := []types.Event{
		newEvent(cmd, "action.requested", map[string]string{
			"user_id":     userID,
			"action_type": payload["action_type"],
			"deadline":    payload["deadline"],
			"prompt":      payload["prompt"],
		}),
	}

	return events, acceptedResult(cmd.CommandID), nil
}

// FIX-14: handleSetTimer emits a timer event for phase deadlines.
func handleSetTimer(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	if cmd.ActorUserID != "autodm" {
		return nil, nil, fmt.Errorf("only autodm can set timers")
	}

	var payload map[string]string
	_ = json.Unmarshal(cmd.Payload, &payload)

	events := []types.Event{
		newEvent(cmd, "timer.set", map[string]string{
			"timer_type": payload["timer_type"],
			"deadline":   payload["deadline"],
		}),
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
		for uid, p := range stateCopy.Players {
			if p.TrueRole == "scarletwoman" && p.Alive {
				if stateCopy.GetAliveCount() >= 5 {
					return []types.Event{
						newEvent(cmd, "demon.changed", map[string]string{
							"old_demon": stateCopy.DemonID,
							"new_demon": uid,
							"reason":    "scarletwoman",
						}),
					}
				}
			}
		}
	}

	return nil
}

func buildGameContext(state State) *game.GameContext {
	// Decide if Recluse registers as evil this night (~50% chance)
	recluseEvil := false
	if n, err := rand.Int(rand.Reader, big.NewInt(2)); err == nil {
		recluseEvil = n.Int64() == 1
	}

	ctx := &game.GameContext{
		Players:             make(map[string]*game.PlayerState),
		SeatOrder:           state.SeatOrder,
		PoisonedIDs:         make(map[string]bool),
		ProtectedIDs:        make(map[string]bool),
		DeadIDs:             make(map[string]bool),
		DemonID:             state.DemonID,
		MinionIDs:           state.MinionIDs,
		NightNumber:         state.NightCount,
		RedHerringID:        state.RedHerringID,
		ExecutedToday:       state.ExecutedToday,
		RecluseRegisterEvil: recluseEvil,
	}

	for uid, p := range state.Players {
		team := game.TeamGood
		if p.Team == "evil" {
			team = game.TeamEvil
		}
		ctx.Players[uid] = &game.PlayerState{
			UserID:          uid,
			SeatNumber:      p.SeatNumber,
			Role:            p.Role,
			TrueRole:        p.TrueRole,
			Team:            team,
			IsAlive:         p.Alive,
			IsPoisoned:      p.IsPoisoned,
			IsProtected:     p.IsProtected,
			SpyApparentRole: p.SpyApparentRole,
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
