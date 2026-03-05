// Package engine 游戏状态结构体定义与事件归约逻辑
//
// [OUT] api（状态查询与回放）
// [OUT] projection（状态脱敏）
// [OUT] room（状态管理与快照）
// [OUT] agent（状态视图）
// [POS] 游戏状态机的数据层，定义状态结构并实现 Reduce 归约
package engine

import (
	"encoding/json"
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
	UserID          string            `json:"user_id"`
	Name            string            `json:"name"`
	SeatNumber      int               `json:"seat_number"`
	Role            string            `json:"role"`      // Perceived role
	TrueRole        string            `json:"true_role"` // Actual role (for drunk)
	Team            string            `json:"team"`      // "good" or "evil"
	Alive           bool              `json:"alive"`
	IsDM            bool              `json:"is_dm"`
	HasGhostVote    bool              `json:"has_ghost_vote"`
	HasNominated    bool              `json:"has_nominated"` // Has nominated today
	WasNominated    bool              `json:"was_nominated"` // Was nominated today
	IsPoisoned      bool              `json:"is_poisoned"`
	IsProtected     bool              `json:"is_protected"`
	ButlerMaster    string            `json:"butler_master,omitempty"`
	SpyApparentRole string            `json:"spy_apparent_role,omitempty"` // 间谍在信息角色面前显示的假身份
	Reminders       []string          `json:"reminders"`
	NightInfo       map[string]string `json:"night_info,omitempty"`
}

type Nomination struct {
	Nominator       string          `json:"nominator"`
	Nominee         string          `json:"nominee"`
	NominatorSeat   int             `json:"nominator_seat"`
	NomineeSeat     int             `json:"nominee_seat"`
	Votes           map[string]bool `json:"votes"`
	VoteOrder       []string        `json:"vote_order"`        // Planned voting sequence (user_ids, clockwise from nominee+1)
	CurrentVoterIdx int             `json:"current_voter_idx"` // Index into VoteOrder for who votes next
	Resolved        bool            `json:"resolved"`
	Result          string          `json:"result"` // "on_the_block", "not_on_the_block", "tied"
	VotesFor        int             `json:"votes_for"`
	VotesAgainst    int             `json:"votes_against"`
	Threshold       int             `json:"threshold"` // Votes needed for execution
	StartedAt       int64           `json:"started_at"`
	DefenseEndsAt   int64           `json:"defense_ends_at"`
	VotingEndsAt    int64           `json:"voting_ends_at"`
}

// OnTheBlockInfo tracks the player currently "about to die" (待处决).
// In official BotC rules, the nominee with the most votes (>= threshold)
// is "on the block". At end of day they are executed unless a subsequent
// nominee gets strictly more votes, or a tie clears the block.
type OnTheBlockInfo struct {
	UserID     string `json:"user_id"`
	VotesFor   int    `json:"votes_for"`
	SeatNumber int    `json:"seat_number"`
}

type NightAction struct {
	UserID     string   `json:"user_id"`
	RoleID     string   `json:"role_id"`
	Order      int      `json:"order"`
	ActionType string   `json:"action_type,omitempty"`
	Completed  bool     `json:"completed"`
	TargetIDs  []string `json:"target_ids,omitempty"`
	Result     string   `json:"result,omitempty"`
}

type PendingDeath struct {
	UserID    string `json:"user_id"`
	Cause     string `json:"cause"` // "demon", "execution", "ability"
	Protected bool   `json:"protected"`
}

type State struct {
	RoomID                string            `json:"room_id"`
	Edition               string            `json:"edition"` // tb, bmr, snv
	MaxPlayers            int               `json:"max_players"`
	Phase                 Phase             `json:"phase"`
	SubPhase              SubPhase          `json:"sub_phase"`
	DayCount              int               `json:"day_count"`
	NightCount            int               `json:"night_count"`
	Players               map[string]Player `json:"players"`
	SeatOrder             []string          `json:"seat_order"` // UserIDs in seat order
	Nomination            *Nomination       `json:"nomination,omitempty"`
	NominationQueue       []Nomination      `json:"nomination_queue"`       // Past nominations today
	OnTheBlock            *OnTheBlockInfo   `json:"on_the_block,omitempty"` // Player about to die
	NightActions          []NightAction     `json:"night_actions"`
	CurrentAction         int               `json:"current_action"` // Index in night actions
	PendingDeaths         []PendingDeath    `json:"pending_deaths"`
	DemonID               string            `json:"demon_id"`
	MinionIDs             []string          `json:"minion_ids"`
	BluffRoles            []string          `json:"bluff_roles"`             // 3 bluffs for demon
	ExecutedToday         string            `json:"executed_today"`          // UserID of player executed today (for undertaker)
	RedHerringID          string            `json:"red_herring_id"`          // Good player that registers as demon to fortune teller
	ScarletWomanTriggered bool              `json:"scarlet_woman_triggered"` // 红唇女郎是否已继承，防重复触发
	AwaitingRavenkeeper   bool              `json:"awaiting_ravenkeeper"`    // 结算层等待守鸦人选择目标
	OwnerID               string            `json:"owner_id,omitempty"`      // First player to join becomes owner
	Winner                string            `json:"winner,omitempty"`        // "good" or "evil"
	WinReason             string            `json:"win_reason,omitempty"`
	ChatSeq               int64             `json:"chat_seq"`
	LastSeq               int64             `json:"last_seq"`
	PhaseStartedAt        int64             `json:"phase_started_at"`
	PhaseEndsAt           int64             `json:"phase_ends_at"`
	ExtensionsUsed        int               `json:"extensions_used"`
	Config                GameConfig        `json:"config"`
	AIDecisionLog         []AIDecisionEntry `json:"ai_decision_log"`
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
	DiscussionDurationSec      int `json:"discussion_duration_sec"`
	NominationTimeoutSec       int `json:"nomination_timeout_sec"`
	DefenseDurationSec         int `json:"defense_duration_sec"`
	VotingDurationSec          int `json:"voting_duration_sec"`
	NightActionTimeoutSec      int `json:"night_action_timeout_sec"`
	ExtensionDurationSec       int `json:"extension_duration_sec"`
	MaxExtensions              int `json:"max_extensions"`
	NominationPhaseDurationSec int `json:"nomination_phase_duration_sec"`
}

func DefaultGameConfig() GameConfig {
	return GameConfig{
		DiscussionDurationSec:      0,
		NominationTimeoutSec:       0,
		DefenseDurationSec:         0,
		VotingDurationSec:          0,
		NightActionTimeoutSec:      0,
		ExtensionDurationSec:       0,
		MaxExtensions:              0,
		NominationPhaseDurationSec: 0,
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

	if s.OnTheBlock != nil {
		otb := *s.OnTheBlock
		cp.OnTheBlock = &otb
	}

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
			Nominator:       s.Nomination.Nominator,
			Nominee:         s.Nomination.Nominee,
			NominatorSeat:   s.Nomination.NominatorSeat,
			NomineeSeat:     s.Nomination.NomineeSeat,
			Votes:           votes,
			VoteOrder:       voteOrder,
			CurrentVoterIdx: s.Nomination.CurrentVoterIdx,
			Resolved:        s.Nomination.Resolved,
			Result:          s.Nomination.Result,
			VotesFor:        s.Nomination.VotesFor,
			VotesAgainst:    s.Nomination.VotesAgainst,
			Threshold:       s.Nomination.Threshold,
			StartedAt:       s.Nomination.StartedAt,
			DefenseEndsAt:   s.Nomination.DefenseEndsAt,
			VotingEndsAt:    s.Nomination.VotingEndsAt,
		}
	}
	return cp
}

// Reduce is defined in state_reduce.go

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
	// Check for Saint execution (via ExecutedToday)
	if s.ExecutedToday != "" {
		if p, ok := s.Players[s.ExecutedToday]; ok && p.TrueRole == "saint" && !p.IsPoisoned {
			return true, "evil", "圣徒被处决"
		}
	}

	// Check if demon is dead
	if demon, ok := s.Players[s.DemonID]; ok && !demon.Alive {
		// Check for Scarlet Woman takeover (5+ players alive)
		aliveCount := s.GetAliveCount()
		hasScarletWoman := false
		for _, p := range s.Players {
			if p.TrueRole == "scarletwoman" && p.Alive {
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
	if aliveCount == 3 && s.ExecutedToday == "" {
		for _, p := range s.Players {
			if p.TrueRole == "mayor" && p.Alive && !p.IsPoisoned {
				return true, "good", "市长在最后三人时达成胜利条件"
			}
		}
	}

	// Check if only 2 players remain - evil wins
	if aliveCount <= 2 {
		return true, "evil", "只剩2名玩家存活"
	}

	return false, "", ""
}
