package engine

import (
	"encoding/json"
	"time"
)

type Phase string

const (
	PhaseLobby      Phase = "lobby"
	PhaseFirstNight Phase = "first_night"
	PhaseDay        Phase = "day"
	PhaseNomination Phase = "nomination"
	PhaseVoting     Phase = "voting"
	PhaseNight      Phase = "night"
	PhaseEnded      Phase = "ended"
)

type SubPhase string

const (
	SubPhaseNone           SubPhase = ""
	SubPhaseDiscussion     SubPhase = "discussion"
	SubPhaseNominationOpen SubPhase = "nomination_open"
	SubPhaseDefense        SubPhase = "defense"
	SubPhaseVoting         SubPhase = "voting"
)

type Player struct {
	UserID       string            `json:"user_id"`
	Name         string            `json:"name"`
	SeatNumber   int               `json:"seat_number"`
	Role         string            `json:"role"`      // Perceived role
	TrueRole     string            `json:"true_role"` // Actual role (for drunk)
	Team         string            `json:"team"`      // "good" or "evil"
	Alive        bool              `json:"alive"`
	IsDM         bool              `json:"is_dm"`
	HasGhostVote bool              `json:"has_ghost_vote"`
	HasNominated bool              `json:"has_nominated"` // Has nominated today
	WasNominated bool              `json:"was_nominated"` // Was nominated today
	IsPoisoned   bool              `json:"is_poisoned"`
	IsProtected  bool              `json:"is_protected"`
	ButlerMaster string            `json:"butler_master,omitempty"`
	Reminders    []string          `json:"reminders"`
	NightInfo    map[string]string `json:"night_info,omitempty"`
}

type Nomination struct {
	Nominator     string          `json:"nominator"`
	Nominee       string          `json:"nominee"`
	NominatorSeat int             `json:"nominator_seat"`
	NomineeSeat   int             `json:"nominee_seat"`
	Votes         map[string]bool `json:"votes"`
	VoteOrder     []string        `json:"vote_order"`
	Resolved      bool            `json:"resolved"`
	Result        string          `json:"result"` // "executed", "not_executed", "tie"
	VotesFor      int             `json:"votes_for"`
	VotesAgainst  int             `json:"votes_against"`
	Threshold     int             `json:"threshold"` // Votes needed for execution
	StartedAt     int64           `json:"started_at"`
	DefenseEndsAt int64           `json:"defense_ends_at"`
	VotingEndsAt  int64           `json:"voting_ends_at"`
}

type NightAction struct {
	UserID    string   `json:"user_id"`
	RoleID    string   `json:"role_id"`
	Order     int      `json:"order"`
	Completed bool     `json:"completed"`
	TargetIDs []string `json:"target_ids,omitempty"`
	Result    string   `json:"result,omitempty"`
}

type PendingDeath struct {
	UserID    string `json:"user_id"`
	Cause     string `json:"cause"` // "demon", "execution", "ability"
	Protected bool   `json:"protected"`
}

type State struct {
	RoomID          string            `json:"room_id"`
	Edition         string            `json:"edition"` // tb, bmr, snv
	MaxPlayers      int               `json:"max_players"`
	Phase           Phase             `json:"phase"`
	SubPhase        SubPhase          `json:"sub_phase"`
	DayCount        int               `json:"day_count"`
	NightCount      int               `json:"night_count"`
	Players         map[string]Player `json:"players"`
	SeatOrder       []string          `json:"seat_order"` // UserIDs in seat order
	Nomination      *Nomination       `json:"nomination,omitempty"`
	NominationQueue []Nomination      `json:"nomination_queue"` // Past nominations today
	NightActions    []NightAction     `json:"night_actions"`
	CurrentAction   int               `json:"current_action"` // Index in night actions
	PendingDeaths   []PendingDeath    `json:"pending_deaths"`
	DemonID         string            `json:"demon_id"`
	MinionIDs       []string          `json:"minion_ids"`
	BluffRoles      []string          `json:"bluff_roles"`      // 3 bluffs for demon
	ExecutedToday   string            `json:"executed_today"`   // UserID of player executed today (for undertaker)
	RedHerringID    string            `json:"red_herring_id"`   // Good player that registers as demon to fortune teller
	Winner          string            `json:"winner,omitempty"` // "good" or "evil"
	WinReason       string            `json:"win_reason,omitempty"`
	ChatSeq         int64             `json:"chat_seq"`
	LastSeq         int64             `json:"last_seq"`
	PhaseStartedAt  int64             `json:"phase_started_at"`
	PhaseEndsAt     int64             `json:"phase_ends_at"`
	Config          GameConfig        `json:"config"`
	AIDecisionLog   []AIDecisionEntry `json:"ai_decision_log"`
}

type AIDecisionEntry struct {
	Night       int    `json:"night"`
	UserID      string `json:"user_id"`
	PlayerName  string `json:"player_name"`
	Role        string `json:"role"`
	Targets     string `json:"targets,omitempty"`
	TrueResult  string `json:"true_result"`
	GivenResult string `json:"given_result"`
	IsPoisoned  bool   `json:"is_poisoned"`
	IsDrunk     bool   `json:"is_drunk"`
	Timestamp   int64  `json:"timestamp"`
}

type GameConfig struct {
	DiscussionDurationSec int `json:"discussion_duration_sec"`
	NominationTimeoutSec  int `json:"nomination_timeout_sec"`
	DefenseDurationSec    int `json:"defense_duration_sec"`
	VotingDurationSec     int `json:"voting_duration_sec"`
	NightActionTimeoutSec int `json:"night_action_timeout_sec"`
}

func DefaultGameConfig() GameConfig {
	return GameConfig{
		DiscussionDurationSec: 180,
		NominationTimeoutSec:  10,
		DefenseDurationSec:    60,
		VotingDurationSec:     3,
		NightActionTimeoutSec: 30,
	}
}

func NewState(roomID string) State {
	return State{
		RoomID:          roomID,
		Phase:           PhaseLobby,
		Edition:         "tb",
		MaxPlayers:      7,
		Players:         make(map[string]Player),
		SeatOrder:       []string{},
		NominationQueue: []Nomination{},
		NightActions:    []NightAction{},
		PendingDeaths:   []PendingDeath{},
		MinionIDs:       []string{},
		BluffRoles:      []string{},
		Config:          DefaultGameConfig(),
		AIDecisionLog:   []AIDecisionEntry{},
	}
}

func (s State) Copy() State {
	cp := s
	cp.Players = make(map[string]Player, len(s.Players))
	for k, v := range s.Players {
		reminders := make([]string, len(v.Reminders))
		copy(reminders, v.Reminders)
		v.Reminders = reminders
		if v.NightInfo != nil {
			nightInfo := make(map[string]string, len(v.NightInfo))
			for nk, nv := range v.NightInfo {
				nightInfo[nk] = nv
			}
			v.NightInfo = nightInfo
		}
		cp.Players[k] = v
	}

	cp.SeatOrder = make([]string, len(s.SeatOrder))
	copy(cp.SeatOrder, s.SeatOrder)

	cp.MinionIDs = make([]string, len(s.MinionIDs))
	copy(cp.MinionIDs, s.MinionIDs)

	cp.BluffRoles = make([]string, len(s.BluffRoles))
	copy(cp.BluffRoles, s.BluffRoles)

	cp.NominationQueue = make([]Nomination, len(s.NominationQueue))
	copy(cp.NominationQueue, s.NominationQueue)

	cp.NightActions = make([]NightAction, len(s.NightActions))
	copy(cp.NightActions, s.NightActions)

	cp.PendingDeaths = make([]PendingDeath, len(s.PendingDeaths))
	copy(cp.PendingDeaths, s.PendingDeaths)

	cp.AIDecisionLog = make([]AIDecisionEntry, len(s.AIDecisionLog))
	copy(cp.AIDecisionLog, s.AIDecisionLog)

	if s.Nomination != nil {
		votes := make(map[string]bool, len(s.Nomination.Votes))
		for k, v := range s.Nomination.Votes {
			votes[k] = v
		}
		voteOrder := make([]string, len(s.Nomination.VoteOrder))
		copy(voteOrder, s.Nomination.VoteOrder)
		cp.Nomination = &Nomination{
			Nominator:     s.Nomination.Nominator,
			Nominee:       s.Nomination.Nominee,
			NominatorSeat: s.Nomination.NominatorSeat,
			NomineeSeat:   s.Nomination.NomineeSeat,
			Votes:         votes,
			VoteOrder:     voteOrder,
			Resolved:      s.Nomination.Resolved,
			Result:        s.Nomination.Result,
			VotesFor:      s.Nomination.VotesFor,
			VotesAgainst:  s.Nomination.VotesAgainst,
			Threshold:     s.Nomination.Threshold,
			StartedAt:     s.Nomination.StartedAt,
			DefenseEndsAt: s.Nomination.DefenseEndsAt,
			VotingEndsAt:  s.Nomination.VotingEndsAt,
		}
	}
	return cp
}

func (s *State) Reduce(event EventPayload) {
	s.LastSeq = event.Seq
	s.ChatSeq++

	switch event.Type {
	case "player.joined":
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
			Role:         "",
			IsDM:         event.Payload["role"] == "dm",
			HasGhostVote: true,
			Reminders:    []string{},
		}
		s.Players[event.Actor] = p
		s.SeatOrder = append(s.SeatOrder, event.Actor)

	case "player.left":
		delete(s.Players, event.Actor)
		for i, uid := range s.SeatOrder {
			if uid == event.Actor {
				s.SeatOrder = append(s.SeatOrder[:i], s.SeatOrder[i+1:]...)
				break
			}
		}

	case "seat.claimed":
		if p, ok := s.Players[event.Actor]; ok {
			if sn, snOk := event.Payload["seat_number"]; snOk {
				if parsed, err := json.Number(sn).Int64(); err == nil {
					p.SeatNumber = int(parsed)
					s.Players[event.Actor] = p
				}
			}
		}

	case "room.settings.changed":
		if ed, ok := event.Payload["edition"]; ok && ed != "" {
			s.Edition = ed
		}
		if mp, ok := event.Payload["max_players"]; ok && mp != "" {
			if parsed, err := json.Number(mp).Int64(); err == nil {
				s.MaxPlayers = int(parsed)
			}
		}

	case "game.started":
		s.Phase = PhaseFirstNight
		s.NightCount = 1
		s.PhaseStartedAt = time.Now().UnixMilli()

	case "role.assigned":
		userID := event.Payload["user_id"]
		if p, ok := s.Players[userID]; ok {
			p.Role = event.Payload["role"]
			p.TrueRole = event.Payload["true_role"]
			if p.TrueRole == "" {
				p.TrueRole = p.Role
			}
			p.Team = event.Payload["team"]
			s.Players[userID] = p

			if event.Payload["is_demon"] == "true" {
				s.DemonID = userID
			}
			if event.Payload["is_minion"] == "true" {
				s.MinionIDs = append(s.MinionIDs, userID)
			}
		}

	case "bluffs.assigned":
		if bluffs, ok := event.Payload["bluffs"]; ok {
			var bluffList []string
			json.Unmarshal([]byte(bluffs), &bluffList)
			s.BluffRoles = bluffList
		}

	case "red_herring.assigned":
		s.RedHerringID = event.Payload["user_id"]

	case "phase.first_night":
		s.Phase = PhaseFirstNight
		s.NightCount = 1
		s.PhaseStartedAt = time.Now().UnixMilli()

	case "phase.night":
		s.Phase = PhaseNight
		s.NightCount++
		s.SubPhase = SubPhaseNone
		s.PhaseStartedAt = time.Now().UnixMilli()
		// Reset daily flags
		for uid, p := range s.Players {
			p.HasNominated = false
			p.WasNominated = false
			p.IsProtected = false
			s.Players[uid] = p
		}

	case "phase.day":
		s.Phase = PhaseDay
		s.DayCount++
		s.SubPhase = SubPhaseDiscussion
		s.PhaseStartedAt = time.Now().UnixMilli()
		s.PhaseEndsAt = time.Now().Add(time.Duration(s.Config.DiscussionDurationSec) * time.Second).UnixMilli()
		s.Nomination = nil
		s.NominationQueue = []Nomination{}
		s.ExecutedToday = ""

	case "phase.nomination":
		s.Phase = PhaseNomination
		s.SubPhase = SubPhaseNominationOpen
		s.PhaseStartedAt = time.Now().UnixMilli()
		s.PhaseEndsAt = time.Now().Add(time.Duration(s.Config.NominationTimeoutSec) * time.Second).UnixMilli()

	case "nomination.created":
		nominatorID := event.Actor
		nomineeID := event.Payload["nominee"]

		nominator := s.Players[nominatorID]
		nominee := s.Players[nomineeID]

		aliveCount := 0
		for _, p := range s.Players {
			if p.Alive {
				aliveCount++
			}
		}
		threshold := (aliveCount / 2) + 1

		now := time.Now().UnixMilli()
		s.Nomination = &Nomination{
			Nominator:     nominatorID,
			Nominee:       nomineeID,
			NominatorSeat: nominator.SeatNumber,
			NomineeSeat:   nominee.SeatNumber,
			Votes:         make(map[string]bool),
			VoteOrder:     []string{},
			Threshold:     threshold,
			StartedAt:     now,
			DefenseEndsAt: now + int64(s.Config.DefenseDurationSec*1000),
		}
		s.SubPhase = SubPhaseDefense

		// Mark players as having nominated/been nominated
		nominator.HasNominated = true
		nominee.WasNominated = true
		s.Players[nominatorID] = nominator
		s.Players[nomineeID] = nominee

	case "defense.ended":
		if s.Nomination != nil {
			s.SubPhase = SubPhaseVoting
			now := time.Now().UnixMilli()
			s.Nomination.VotingEndsAt = now + int64(s.Config.VotingDurationSec*1000*len(s.Players))
		}

	case "vote.cast":
		if s.Nomination != nil {
			vote := event.Payload["vote"] == "yes"
			s.Nomination.Votes[event.Actor] = vote
			s.Nomination.VoteOrder = append(s.Nomination.VoteOrder, event.Actor)

			if vote {
				s.Nomination.VotesFor++
			} else {
				s.Nomination.VotesAgainst++
			}

			// Check if ghost vote was used
			if p, ok := s.Players[event.Actor]; ok && !p.Alive && vote {
				p.HasGhostVote = false
				s.Players[event.Actor] = p
			}
		}

	case "nomination.resolved":
		if s.Nomination != nil {
			s.Nomination.Resolved = true
			s.Nomination.Result = event.Payload["result"]
			s.NominationQueue = append(s.NominationQueue, *s.Nomination)
			s.SubPhase = SubPhaseNominationOpen
			s.PhaseEndsAt = time.Now().Add(time.Duration(s.Config.NominationTimeoutSec) * time.Second).UnixMilli()
		}

	case "execution.resolved":
		if event.Payload["result"] == "executed" {
			executedID := event.Payload["executed"]
			s.ExecutedToday = executedID
			if p, ok := s.Players[executedID]; ok {
				p.Alive = false
				s.Players[executedID] = p
			}
		}

	case "player.died":
		diedID := event.Payload["user_id"]
		if p, ok := s.Players[diedID]; ok {
			p.Alive = false
			s.Players[diedID] = p
		}

	case "player.protected":
		protectedID := event.Payload["user_id"]
		if p, ok := s.Players[protectedID]; ok {
			p.IsProtected = true
			s.Players[protectedID] = p
		}

	case "player.poisoned":
		poisonedID := event.Payload["user_id"]
		if p, ok := s.Players[poisonedID]; ok {
			p.IsPoisoned = true
			s.Players[poisonedID] = p
		}

	case "poison.cleared":
		for uid, p := range s.Players {
			p.IsPoisoned = false
			s.Players[uid] = p
		}

	case "night.action.queued":
		action := NightAction{
			UserID: event.Payload["user_id"],
			RoleID: event.Payload["role_id"],
		}
		if orderStr, ok := event.Payload["order"]; ok {
			if parsed, err := json.Number(orderStr).Int64(); err == nil {
				action.Order = int(parsed)
			}
		}
		s.NightActions = append(s.NightActions, action)

	case "night.action.completed":
		actionUserID := event.Payload["user_id"]
		for i, a := range s.NightActions {
			if a.UserID == actionUserID && !a.Completed {
				s.NightActions[i].Completed = true
				if targets, ok := event.Payload["targets"]; ok {
					var targetList []string
					json.Unmarshal([]byte(targets), &targetList)
					s.NightActions[i].TargetIDs = targetList
				}
				s.NightActions[i].Result = event.Payload["result"]
				break
			}
		}
		s.CurrentAction++

	case "ability.resolved":
		// Additional ability handling if needed

	case "demon.changed":
		newDemonID := event.Payload["new_demon"]
		oldDemonID := event.Payload["old_demon"]
		s.DemonID = newDemonID

		// Update minion list
		for i, mid := range s.MinionIDs {
			if mid == newDemonID {
				s.MinionIDs = append(s.MinionIDs[:i], s.MinionIDs[i+1:]...)
				break
			}
		}
		s.MinionIDs = append(s.MinionIDs, oldDemonID)

	case "public.chat", "whisper.sent", "evil_team.chat":
		// Just increment chat seq

	case "ai.decision":
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

	case "reminder.added":
		if uid, ok := event.Payload["user_id"]; ok {
			if p, pOk := s.Players[uid]; pOk {
				if reminder, rOk := event.Payload["reminder"]; rOk {
					p.Reminders = append(p.Reminders, reminder)
					s.Players[uid] = p
				}
			}
		}

	case "game.ended":
		s.Phase = PhaseEnded
		s.Winner = event.Payload["winner"]
		s.WinReason = event.Payload["reason"]
	}
}

type EventPayload struct {
	Seq     int64
	Type    string
	Actor   string
	Payload map[string]string
}

func MarshalState(s State) (string, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func UnmarshalState(raw string) (State, error) {
	var s State
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return State{}, err
	}
	return s, nil
}

// GetAliveCount returns the number of alive players.
func (s *State) GetAliveCount() int {
	count := 0
	for _, p := range s.Players {
		if p.Alive && !p.IsDM {
			count++
		}
	}
	return count
}

// GetAliveNeighbors returns the alive neighbors for a player.
func (s *State) GetAliveNeighbors(userID string) (left, right string) {
	idx := -1
	for i, uid := range s.SeatOrder {
		if uid == userID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return "", ""
	}

	// Find left neighbor
	for i := 1; i < len(s.SeatOrder); i++ {
		leftIdx := (idx - i + len(s.SeatOrder)) % len(s.SeatOrder)
		leftUID := s.SeatOrder[leftIdx]
		if p, ok := s.Players[leftUID]; ok && p.Alive {
			left = leftUID
			break
		}
	}

	// Find right neighbor
	for i := 1; i < len(s.SeatOrder); i++ {
		rightIdx := (idx + i) % len(s.SeatOrder)
		rightUID := s.SeatOrder[rightIdx]
		if p, ok := s.Players[rightUID]; ok && p.Alive {
			right = rightUID
			break
		}
	}

	return left, right
}

// CheckWinCondition checks if the game has ended.
func (s *State) CheckWinCondition() (ended bool, winner, reason string) {
	// Check for Saint execution
	for _, nom := range s.NominationQueue {
		if nom.Result == "executed" {
			if p, ok := s.Players[nom.Nominee]; ok && p.TrueRole == "saint" && !p.IsPoisoned {
				return true, "evil", "圣徒被处决"
			}
		}
	}

	// Check if demon is dead
	if demon, ok := s.Players[s.DemonID]; ok && !demon.Alive {
		// Check for Scarlet Woman takeover (5+ players alive)
		aliveCount := s.GetAliveCount()
		hasScarletWoman := false
		for _, p := range s.Players {
			if p.TrueRole == "scarlet_woman" && p.Alive {
				hasScarletWoman = true
				break
			}
		}
		if !hasScarletWoman || aliveCount < 5 {
			return true, "good", "恶魔已死亡"
		}
	}

	// Mayor win: exactly 3 alive, no execution today, mayor alive and not poisoned
	aliveCount := s.GetAliveCount()
	if aliveCount == 3 {
		hasExecutionToday := false
		for _, nom := range s.NominationQueue {
			if nom.Result == "executed" {
				hasExecutionToday = true
				break
			}
		}
		if !hasExecutionToday {
			for _, p := range s.Players {
				if p.TrueRole == "mayor" && p.Alive && !p.IsPoisoned {
					return true, "good", "市长在最后三人时达成胜利条件"
				}
			}
		}
	}

	// Check if only 2 players remain - evil wins
	if aliveCount <= 2 {
		return true, "evil", "只剩2名玩家存活"
	}

	return false, "", ""
}
