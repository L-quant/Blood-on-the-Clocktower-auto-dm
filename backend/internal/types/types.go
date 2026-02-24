package types

import (
	"encoding/json"
	"errors"
	"fmt"
)

type ErrorCode string

const (
	ErrUnauthorized ErrorCode = "unauthorized"
	ErrForbidden    ErrorCode = "forbidden"
	ErrBadRequest   ErrorCode = "bad_request"
	ErrConflict     ErrorCode = "conflict"
	ErrInternal     ErrorCode = "internal"
	ErrNotFound     ErrorCode = "not_found"
	ErrRateLimited  ErrorCode = "rate_limited"
)

type AppError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Err     error     `json:"-"`
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %s", e.Message, e.Err.Error())
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Err }

func NewError(code ErrorCode, msg string) *AppError {
	return &AppError{Code: code, Message: msg}
}

func WrapError(code ErrorCode, msg string, err error) *AppError {
	return &AppError{Code: code, Message: msg, Err: err}
}

func Is(err error, code ErrorCode) bool {
	var app *AppError
	if errors.As(err, &app) {
		return app.Code == code
	}
	return false
}

type CommandEnvelope struct {
	CommandID      string          `json:"command_id"`
	IdempotencyKey string          `json:"idempotency_key"`
	RoomID         string          `json:"room_id"`
	Type           string          `json:"type"`
	LastSeenSeq    int64           `json:"last_seen_seq"`
	ActorUserID    string          `json:"actor_user_id"`
	Payload        json.RawMessage `json:"data"`
}

type Event struct {
	RoomID            string          `json:"room_id"`
	Seq               int64           `json:"seq"`
	EventID           string          `json:"event_id"`
	EventType         string          `json:"event_type"`
	ActorUserID       string          `json:"actor_user_id"`
	CausationCommand  string          `json:"causation_command_id"`
	Payload           json.RawMessage `json:"payload"`
	ServerTimestampMs int64           `json:"server_ts_ms"`
}

type CommandResult struct {
	CommandID      string `json:"command_id"`
	Status         string `json:"status"`
	Reason         string `json:"reason,omitempty"`
	AppliedSeqFrom int64  `json:"applied_seq_from"`
	AppliedSeqTo   int64  `json:"applied_seq_to"`
}

type ProjectedEvent struct {
	RoomID      string          `json:"room_id"`
	Seq         int64           `json:"seq"`
	EventType   string          `json:"event_type"`
	ActorUserID string          `json:"actor_user_id,omitempty"`
	Data        json.RawMessage `json:"data"`
	ServerTS    int64           `json:"server_ts"`
}

type Viewer struct {
	UserID string
	Role   string
	IsDM   bool
}
