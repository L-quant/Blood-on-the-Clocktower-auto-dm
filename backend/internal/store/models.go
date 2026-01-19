package store

import (
	"time"
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

type Room struct {
	ID        string
	CreatedBy string
	DMUserID  string
	Status    string
	CreatedAt time.Time
}

type RoomMember struct {
	RoomID string
	UserID string
	Role   string
	Joined time.Time
}

type DedupRecord struct {
	RoomID         string
	ActorUserID    string
	IdempotencyKey string
	CommandType    string
	CommandID      string
	Status         string
	ResultJSON     string
	CreatedAt      time.Time
}

type Snapshot struct {
	RoomID    string
	LastSeq   int64
	StateJSON string
	CreatedAt time.Time
}

type AgentRun struct {
	ID           string
	RoomID       string
	SeqFrom      int64
	SeqTo        int64
	AgentName    string
	ViewerUserID *string
	InputDigest  string
	OutputDigest string
	Status       string
	LatencyMs    int64
	ErrorText    string
	CreatedAt    time.Time
}
