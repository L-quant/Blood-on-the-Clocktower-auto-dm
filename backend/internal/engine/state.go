package engine

import (
	"encoding/json"
)

type Phase string

const (
	PhaseLobby Phase = "lobby"
	PhaseDay   Phase = "day"
	PhaseNight Phase = "night"
	PhaseEnded Phase = "ended"
)

type Player struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	Alive  bool   `json:"alive"`
	IsDM   bool   `json:"is_dm"`
}

type Nomination struct {
	Nominator string          `json:"nominator"`
	Nominee   string          `json:"nominee"`
	Votes     map[string]bool `json:"votes"`
	Resolved  bool            `json:"resolved"`
	Result    string          `json:"result"`
}

type State struct {
	RoomID     string            `json:"room_id"`
	Phase      Phase             `json:"phase"`
	DayCount   int               `json:"day_count"`
	NightCount int               `json:"night_count"`
	Players    map[string]Player `json:"players"`
	Nomination *Nomination       `json:"nomination,omitempty"`
	ChatSeq    int64             `json:"chat_seq"`
	LastSeq    int64             `json:"last_seq"`
}

func NewState(roomID string) State {
	return State{
		RoomID:     roomID,
		Phase:      PhaseLobby,
		Players:    make(map[string]Player),
		Nomination: nil,
	}
}

func (s State) Copy() State {
	cp := s
	cp.Players = make(map[string]Player, len(s.Players))
	for k, v := range s.Players {
		cp.Players[k] = v
	}
	if s.Nomination != nil {
		votes := make(map[string]bool, len(s.Nomination.Votes))
		for k, v := range s.Nomination.Votes {
			votes[k] = v
		}
		cp.Nomination = &Nomination{
			Nominator: s.Nomination.Nominator,
			Nominee:   s.Nomination.Nominee,
			Votes:     votes,
			Resolved:  s.Nomination.Resolved,
			Result:    s.Nomination.Result,
		}
	}
	return cp
}

func (s *State) Reduce(event EventPayload) {
	s.LastSeq = event.Seq
	s.ChatSeq++
	switch event.Type {
	case "player.joined":
		p := Player{UserID: event.Actor, Alive: true, Role: "", IsDM: event.Payload["role"] == "dm"}
		s.Players[event.Actor] = p
	case "player.left":
		delete(s.Players, event.Actor)
	case "game.started":
		s.Phase = PhaseDay
		s.DayCount = 1
	case "phase.night":
		s.Phase = PhaseNight
		s.NightCount++
	case "phase.day":
		s.Phase = PhaseDay
		s.DayCount++
	case "nomination.created":
		s.Nomination = &Nomination{Nominator: event.Actor, Nominee: event.Payload["nominee"], Votes: map[string]bool{}}
	case "vote.cast":
		if s.Nomination != nil {
			s.Nomination.Votes[event.Actor] = event.Payload["vote"] == "yes"
		}
	case "execution.resolved":
		if s.Nomination != nil {
			s.Nomination.Resolved = true
			s.Nomination.Result = event.Payload["result"]
			if event.Payload["result"] == "executed" {
				if pl, ok := s.Players[s.Nomination.Nominee]; ok {
					pl.Alive = false
					s.Players[s.Nomination.Nominee] = pl
				}
			}
		}
	case "ability.resolved":
	case "game.ended":
		s.Phase = PhaseEnded
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
