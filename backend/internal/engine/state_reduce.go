// Package engine 事件归约逻辑：将事件应用到游戏状态
//
// [OUT] state.go（State 结构体定义）
// [POS] Reduce 方法处理 30+ 种事件类型，更新游戏状态
package engine

import (
	"encoding/json"
	"time"
)

// Reduce applies an event to the state.
func (s *State) Reduce(event EventPayload) {
	s.LastSeq = event.Seq
	s.ChatSeq++

	switch event.Type {
	case "player.joined":
		s.reducePlayerJoined(event)
	case "player.left":
		s.reducePlayerLeft(event)
	case "seat.claimed":
		s.reduceSeatClaimed(event)
	case "room.settings.changed":
		s.reduceRoomSettings(event)
	case "game.started":
		s.Phase = PhaseFirstNight
		s.NightCount = 1
		s.PhaseStartedAt = time.Now().UnixMilli()
	case "role.assigned":
		s.reduceRoleAssigned(event)
	case "bluffs.assigned":
		s.reduceBluffsAssigned(event)
	case "red_herring.assigned":
		s.RedHerringID = event.Payload["user_id"]
	case "phase.first_night":
		s.Phase = PhaseFirstNight
		s.NightCount = 1
		s.PhaseStartedAt = time.Now().UnixMilli()
	case "phase.night":
		s.reducePhaseNight()
	case "phase.day":
		s.reducePhaseDay()
	case "phase.nomination":
		s.Phase = PhaseNomination
		s.SubPhase = SubPhaseNominationOpen
		s.PhaseStartedAt = time.Now().UnixMilli()
		s.PhaseEndsAt = time.Now().Add(time.Duration(s.Config.NominationTimeoutSec) * time.Second).UnixMilli()
	case "nomination.created":
		s.reduceNominationCreated(event)
	case "defense.progress":
		s.reduceDefenseProgress(event)
	case "defense.ended":
		s.reduceDefenseEnded()
	case "vote.cast":
		s.reduceVoteCast(event)
	case "nomination.resolved":
		s.reduceNominationResolved(event)
	case "execution.resolved":
		s.reduceExecutionResolved(event)
	case "player.died":
		s.reducePlayerDied(event.Payload["user_id"])
	case "player.protected":
		s.reducePlayerFlag(event.Payload["user_id"], "protected")
	case "player.poisoned":
		s.reducePlayerFlag(event.Payload["user_id"], "poisoned")
	case "poison.cleared":
		for uid, p := range s.Players {
			p.IsPoisoned = false
			s.Players[uid] = p
		}
	case "night.action.queued":
		s.reduceNightActionQueued(event)
	case "night.action.completed":
		s.reduceNightActionCompleted(event)
	case "night.action.prompt":
		// No-op: prompt is a signal to the frontend, no state change needed
	case "ability.resolved":
		// Additional ability handling if needed
	case "night.info":
		s.reduceNightInfo(event)
	case "team.recognition":
		// No-op: informational event for frontend — no state mutation
	case "poison.rollback":
		s.reducePlayerUnpoison(event.Payload["user_id"])
	case "demon.changed":
		s.reduceDemonChanged(event)
	case "public.chat", "whisper.sent", "evil_team.chat":
		// Just increment chat seq
	case "ai.decision":
		s.reduceAIDecision(event)
	case "reminder.added":
		s.reduceReminderAdded(event)
	case "game.ended":
		s.Phase = PhaseEnded
		s.Winner = event.Payload["winner"]
		s.WinReason = event.Payload["reason"]
	case "game.recap":
		s.GameRecap = event.Payload["summary"]
	case "player.executed":
		executedID := event.Payload["user_id"]
		s.ExecutedToday = executedID
		s.reducePlayerDied(executedID)
	case "action.requested":
		// informational, no state mutation
	case "timer.set":
		if deadlineStr, ok := event.Payload["deadline"]; ok {
			if deadline, err := json.Number(deadlineStr).Int64(); err == nil {
				s.PhaseEndsAt = deadline
			}
		}
	case "time.extended":
		s.ExtensionsUsed++
		if deadlineStr, ok := event.Payload["deadline"]; ok {
			if deadline, err := json.Number(deadlineStr).Int64(); err == nil {
				s.PhaseEndsAt = deadline
			}
		}
	case "slayer.shot":
		// informational, death handled by player.died
	}
}

func (s *State) reducePlayerJoined(event EventPayload) {
	seatNum := len(s.Players) + 1
	if sn, ok := event.Payload["seat_number"]; ok {
		if parsed, err := json.Number(sn).Int64(); err == nil {
			seatNum = int(parsed)
		}
	}
	p := Player{
		UserID:       event.Actor,
		Name:         event.Payload["name"],
		SeatNumber:   seatNum,
		Alive:        true,
		IsDM:         event.Payload["role"] == "dm",
		HasGhostVote: true,
		Reminders:    []string{},
	}
	s.Players[event.Actor] = p
	s.SeatOrder = append(s.SeatOrder, event.Actor)
	if s.OwnerID == "" && !p.IsDM {
		s.OwnerID = event.Actor
	}
}

func (s *State) reducePlayerLeft(event EventPayload) {
	delete(s.Players, event.Actor)
	for i, uid := range s.SeatOrder {
		if uid == event.Actor {
			s.SeatOrder = append(s.SeatOrder[:i], s.SeatOrder[i+1:]...)
			break
		}
	}
	if s.OwnerID == event.Actor {
		s.OwnerID = ""
		for _, uid := range s.SeatOrder {
			if p, ok := s.Players[uid]; ok && !p.IsDM {
				s.OwnerID = uid
				break
			}
		}
	}
}

func (s *State) reduceSeatClaimed(event EventPayload) {
	if p, ok := s.Players[event.Actor]; ok {
		if sn, snOk := event.Payload["seat_number"]; snOk {
			if parsed, err := json.Number(sn).Int64(); err == nil {
				p.SeatNumber = int(parsed)
				s.Players[event.Actor] = p
			}
		}
	}
}

func (s *State) reduceRoomSettings(event EventPayload) {
	if ed, ok := event.Payload["edition"]; ok && ed != "" {
		s.Edition = ed
	}
	if mp, ok := event.Payload["max_players"]; ok && mp != "" {
		if parsed, err := json.Number(mp).Int64(); err == nil {
			s.MaxPlayers = int(parsed)
		}
	}
}

func (s *State) reduceRoleAssigned(event EventPayload) {
	userID := event.Payload["user_id"]
	p, ok := s.Players[userID]
	if !ok {
		return
	}
	p.Role = event.Payload["role"]
	p.TrueRole = event.Payload["true_role"]
	if p.TrueRole == "" {
		p.TrueRole = p.Role
	}
	p.Team = event.Payload["team"]
	if sar, ok := event.Payload["spy_apparent_role"]; ok && sar != "" {
		p.SpyApparentRole = sar
	}
	s.Players[userID] = p
	if event.Payload["is_demon"] == "true" {
		s.DemonID = userID
	}
	if event.Payload["is_minion"] == "true" {
		s.MinionIDs = append(s.MinionIDs, userID)
	}
}

func (s *State) reduceBluffsAssigned(event EventPayload) {
	bluffs, ok := event.Payload["bluffs"]
	if !ok {
		return
	}
	var bluffList []string
	if err := json.Unmarshal([]byte(bluffs), &bluffList); err == nil {
		s.BluffRoles = bluffList
	}
}

func (s *State) reducePhaseNight() {
	s.Phase = PhaseNight
	s.NightCount++
	s.SubPhase = SubPhaseNone
	s.PhaseStartedAt = time.Now().UnixMilli()
	s.NightActions = []NightAction{}
	s.CurrentAction = 0
	s.PendingDeaths = []PendingDeath{}
	for uid, p := range s.Players {
		p.HasNominated = false
		p.WasNominated = false
		p.IsProtected = false
		s.Players[uid] = p
	}
}

func (s *State) reducePhaseDay() {
	s.Phase = PhaseDay
	s.DayCount++
	s.SubPhase = SubPhaseDiscussion
	s.PhaseStartedAt = time.Now().UnixMilli()
	s.PhaseEndsAt = time.Now().Add(time.Duration(s.Config.DiscussionDurationSec) * time.Second).UnixMilli()
	s.Nomination = nil
	s.NominationQueue = []Nomination{}
	s.OnTheBlock = nil
	s.ExecutedToday = ""
	s.ExtensionsUsed = 0
}

// buildVoteOrder produces the sequential voting list (user_ids) starting
// from the seat after nomineeSeat, clockwise, including only eligible voters.
func (s *State) buildVoteOrder(nomineeSeat int) []string {
	n := len(s.SeatOrder)
	if n == 0 {
		return []string{}
	}
	nomineeIdx := -1
	for i, uid := range s.SeatOrder {
		if s.Players[uid].SeatNumber == nomineeSeat {
			nomineeIdx = i
			break
		}
	}
	if nomineeIdx < 0 {
		return []string{}
	}
	order := make([]string, 0, n)
	for offset := 1; offset <= n; offset++ {
		idx := (nomineeIdx + offset) % n
		uid := s.SeatOrder[idx]
		p := s.Players[uid]
		if p.Alive || p.HasGhostVote {
			order = append(order, uid)
		}
	}
	return order
}

func (s *State) reduceNominationCreated(event EventPayload) {
	nominatorID := event.Actor
	if nuid, ok := event.Payload["nominator_user_id"]; ok && nuid != "" {
		nominatorID = nuid
	}
	nomineeID := event.Payload["nominee"]
	nominator := s.Players[nominatorID]
	nominee := s.Players[nomineeID]

	aliveCount := 0
	for _, p := range s.Players {
		if p.Alive {
			aliveCount++
		}
	}
	threshold := (aliveCount + 1) / 2
	now := time.Now().UnixMilli()
	s.Nomination = &Nomination{
		Nominator:      nominatorID,
		Nominee:        nomineeID,
		NominatorSeat:  nominator.SeatNumber,
		NomineeSeat:    nominee.SeatNumber,
		Votes:          make(map[string]bool),
		VoteOrder:      s.buildVoteOrder(nominee.SeatNumber),
		Threshold:      threshold,
		StartedAt:      now,
		DefenseEndsAt:  now + int64(s.Config.DefenseDurationSec*1000),
		NominatorEnded: false,
		NomineeEnded:   false,
	}
	s.SubPhase = SubPhaseDefense
	// FIX: Handle self-nomination — when nominator == nominee, both flags
	// must be set on the same struct copy to avoid overwrite.
	if nominatorID == nomineeID {
		nominator.HasNominated = true
		nominator.WasNominated = true
		s.Players[nominatorID] = nominator
	} else {
		nominator.HasNominated = true
		nominee.WasNominated = true
		s.Players[nominatorID] = nominator
		s.Players[nomineeID] = nominee
	}
}

func (s *State) reduceDefenseProgress(event EventPayload) {
	if s.Nomination == nil {
		return
	}
	uid := event.Payload["user_id"]
	if uid == s.Nomination.Nominator {
		s.Nomination.NominatorEnded = true
	}
	if uid == s.Nomination.Nominee {
		s.Nomination.NomineeEnded = true
	}
}

func (s *State) reduceDefenseEnded() {
	if s.Nomination == nil {
		return
	}
	s.SubPhase = SubPhaseVoting
	now := time.Now().UnixMilli()
	s.Nomination.VotingEndsAt = now + int64(s.Config.VotingDurationSec*1000*len(s.Players))
}

func (s *State) reduceVoteCast(event EventPayload) {
	if s.Nomination == nil {
		return
	}
	vote := event.Payload["vote"] == "yes"
	s.Nomination.Votes[event.Actor] = vote
	if vote {
		s.Nomination.VotesFor++
	} else {
		s.Nomination.VotesAgainst++
	}
	// Advance sequential voter index
	s.Nomination.CurrentVoterIdx++
	if p, ok := s.Players[event.Actor]; ok && !p.Alive && vote {
		p.HasGhostVote = false
		s.Players[event.Actor] = p
	}
}

func (s *State) reduceNominationResolved(event EventPayload) {
	if s.Nomination == nil {
		return
	}
	s.Nomination.Resolved = true
	result := event.Payload["result"]
	s.Nomination.Result = result
	s.NominationQueue = append(s.NominationQueue, *s.Nomination)
	s.SubPhase = SubPhaseNominationOpen
	s.PhaseEndsAt = time.Now().Add(time.Duration(s.Config.NominationPhaseDurationSec) * time.Second).UnixMilli()

	// On-the-block logic: track the nominee with the most votes
	votesFor := 0
	if vf, ok := event.Payload["votes_for"]; ok {
		if parsed, err := json.Number(vf).Int64(); err == nil {
			votesFor = int(parsed)
		}
	}
	switch result {
	case "on_the_block":
		nominee := s.Players[s.Nomination.Nominee]
		s.OnTheBlock = &OnTheBlockInfo{
			UserID:     s.Nomination.Nominee,
			VotesFor:   votesFor,
			SeatNumber: nominee.SeatNumber,
		}
	case "tied":
		s.OnTheBlock = nil // Tie clears the block — no execution
	}
}

func (s *State) reduceExecutionResolved(event EventPayload) {
	if event.Payload["result"] != "executed" {
		return
	}
	executedID := event.Payload["executed"]
	s.ExecutedToday = executedID
	s.reducePlayerDied(executedID)
}

func (s *State) reducePlayerDied(userID string) {
	if p, ok := s.Players[userID]; ok {
		p.Alive = false
		s.Players[userID] = p
	}
}

func (s *State) reducePlayerFlag(userID string, flag string) {
	p, ok := s.Players[userID]
	if !ok {
		return
	}
	switch flag {
	case "protected":
		p.IsProtected = true
	case "poisoned":
		p.IsPoisoned = true
	}
	s.Players[userID] = p
}

func (s *State) reduceNightActionQueued(event EventPayload) {
	action := NightAction{
		UserID:     event.Payload["user_id"],
		RoleID:     event.Payload["role_id"],
		ActionType: event.Payload["action_type"],
	}
	if orderStr, ok := event.Payload["order"]; ok {
		if parsed, err := json.Number(orderStr).Int64(); err == nil {
			action.Order = int(parsed)
		}
	}
	s.NightActions = append(s.NightActions, action)
}

func (s *State) reduceNightActionCompleted(event EventPayload) {
	actionUserID := event.Payload["user_id"]
	for i, a := range s.NightActions {
		if a.UserID == actionUserID && !a.Completed {
			s.NightActions[i].Completed = true
			if targets, ok := event.Payload["targets"]; ok {
				var targetList []string
				if err := json.Unmarshal([]byte(targets), &targetList); err == nil {
					s.NightActions[i].TargetIDs = targetList
				}
			}
			s.NightActions[i].Result = event.Payload["result"]
			break
		}
	}
	// Recalculate CurrentAction: index of first uncompleted action
	s.CurrentAction = len(s.NightActions)
	for i, a := range s.NightActions {
		if !a.Completed {
			s.CurrentAction = i
			break
		}
	}
}

func (s *State) reduceDemonChanged(event EventPayload) {
	newDemonID := event.Payload["new_demon"]
	oldDemonID := event.Payload["old_demon"]
	s.DemonID = newDemonID
	for i, mid := range s.MinionIDs {
		if mid == newDemonID {
			s.MinionIDs = append(s.MinionIDs[:i], s.MinionIDs[i+1:]...)
			break
		}
	}
	s.MinionIDs = append(s.MinionIDs, oldDemonID)
}

func (s *State) reduceAIDecision(event EventPayload) {
	night := 0
	if n, ok := event.Payload["night"]; ok {
		if parsed, err := json.Number(n).Int64(); err == nil {
			night = int(parsed)
		}
	}
	var ts int64
	if t, ok := event.Payload["timestamp"]; ok {
		if parsed, err := json.Number(t).Int64(); err == nil {
			ts = parsed
		}
	}
	entry := AIDecisionEntry{
		Night:       night,
		UserID:      event.Payload["user_id"],
		PlayerName:  event.Payload["player_name"],
		Role:        event.Payload["role"],
		Targets:     event.Payload["targets"],
		TrueResult:  event.Payload["true_result"],
		GivenResult: event.Payload["given_result"],
		IsPoisoned:  event.Payload["is_poisoned"] == "true",
		IsDrunk:     event.Payload["is_drunk"] == "true",
		Timestamp:   ts,
	}
	s.AIDecisionLog = append(s.AIDecisionLog, entry)
}

func (s *State) reduceReminderAdded(event EventPayload) {
	uid, ok := event.Payload["user_id"]
	if !ok {
		return
	}
	p, pOk := s.Players[uid]
	if !pOk {
		return
	}
	if reminder, rOk := event.Payload["reminder"]; rOk {
		p.Reminders = append(p.Reminders, reminder)
		s.Players[uid] = p
	}
}

func (s *State) reduceNightInfo(event EventPayload) {
	uid, ok := event.Payload["user_id"]
	if !ok {
		return
	}
	p, pOk := s.Players[uid]
	if !pOk {
		return
	}
	if p.NightInfo == nil {
		p.NightInfo = make(map[string]string)
	}
	p.NightInfo["info_type"] = event.Payload["info_type"]
	p.NightInfo["content"] = event.Payload["content"]
	p.NightInfo["message"] = event.Payload["message"]
	if isFalse, ok := event.Payload["is_false"]; ok {
		p.NightInfo["is_false"] = isFalse
	}
	s.Players[uid] = p
}

func (s *State) reducePlayerUnpoison(userID string) {
	if p, ok := s.Players[userID]; ok {
		p.IsPoisoned = false
		s.Players[userID] = p
	}
}
