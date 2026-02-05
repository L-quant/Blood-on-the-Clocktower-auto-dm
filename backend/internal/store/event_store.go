package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type StoredEvent struct {
	RoomID           string
	Seq              int64
	EventID          string
	EventType        string
	ActorUserID      string
	CausationCommand string
	PayloadJSON      string
	ServerTime       time.Time
}

func (s *Store) GetDedupRecord(ctx context.Context, roomID, actorUserID, idempotencyKey, commandType string) (*DedupRecord, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT room_id,actor_user_id,idempotency_key,command_type,command_id,status,result_json,created_at FROM commands_dedup WHERE room_id=? AND actor_user_id=? AND idempotency_key=? AND command_type=?`, roomID, actorUserID, idempotencyKey, commandType)
	var r DedupRecord
	if err := row.Scan(&r.RoomID, &r.ActorUserID, &r.IdempotencyKey, &r.CommandType, &r.CommandID, &r.Status, &r.ResultJSON, &r.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

func (s *Store) SaveDedupRecord(ctx context.Context, tx *sql.Tx, r DedupRecord) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO commands_dedup (room_id,actor_user_id,idempotency_key,command_type,command_id,status,result_json,created_at) VALUES (?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE status=VALUES(status),result_json=VALUES(result_json)`,
		r.RoomID, r.ActorUserID, r.IdempotencyKey, r.CommandType, r.CommandID, r.Status, r.ResultJSON, r.CreatedAt)
	return err
}

func (s *Store) GetLatestSnapshot(ctx context.Context, roomID string) (*Snapshot, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT room_id,last_seq,state_json,created_at FROM snapshots WHERE room_id=? ORDER BY last_seq DESC LIMIT 1`, roomID)
	var snap Snapshot
	if err := row.Scan(&snap.RoomID, &snap.LastSeq, &snap.StateJSON, &snap.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &snap, nil
}

func (s *Store) SaveSnapshot(ctx context.Context, tx *sql.Tx, snap Snapshot) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO snapshots (room_id,last_seq,state_json,created_at) VALUES (?,?,?,?)`, snap.RoomID, snap.LastSeq, snap.StateJSON, snap.CreatedAt)
	return err
}

func (s *Store) LoadEventsAfter(ctx context.Context, roomID string, afterSeq int64, limit int) ([]StoredEvent, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT room_id,seq,event_id,event_type,actor_user_id,causation_command_id,payload_json,server_ts FROM events WHERE room_id=? AND seq>? ORDER BY seq ASC LIMIT ?`, roomID, afterSeq, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []StoredEvent
	for rows.Next() {
		var e StoredEvent
		if err := rows.Scan(&e.RoomID, &e.Seq, &e.EventID, &e.EventType, &e.ActorUserID, &e.CausationCommand, &e.PayloadJSON, &e.ServerTime); err != nil {
			return nil, err
		}
		res = append(res, e)
	}
	return res, rows.Err()
}

func (s *Store) LoadEventsUpTo(ctx context.Context, roomID string, toSeq int64) ([]StoredEvent, error) {
	var (
		rows *sql.Rows
		err  error
	)

	if toSeq > 0 {
		rows, err = s.DB.QueryContext(ctx,
			`SELECT room_id,seq,event_id,event_type,actor_user_id,causation_command_id,payload_json,server_ts
			 FROM events WHERE room_id=? AND seq<=? ORDER BY seq ASC`,
			roomID, toSeq)
	} else {
		rows, err = s.DB.QueryContext(ctx,
			`SELECT room_id,seq,event_id,event_type,actor_user_id,causation_command_id,payload_json,server_ts
			 FROM events WHERE room_id=? ORDER BY seq ASC`,
			roomID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []StoredEvent
	for rows.Next() {
		var e StoredEvent
		if err := rows.Scan(&e.RoomID, &e.Seq, &e.EventID, &e.EventType, &e.ActorUserID, &e.CausationCommand, &e.PayloadJSON, &e.ServerTime); err != nil {
			return nil, err
		}
		res = append(res, e)
	}
	return res, rows.Err()
}

func (s *Store) AppendEvents(ctx context.Context, roomID string, events []StoredEvent, dedup *DedupRecord, snap *Snapshot) error {
	return s.WithTx(ctx, func(tx *sql.Tx) error {
		var current int64
		row := tx.QueryRowContext(ctx, `SELECT next_seq FROM room_sequences WHERE room_id=? FOR UPDATE`, roomID)
		switch err := row.Scan(&current); err {
		case nil:
		case sql.ErrNoRows:
			current = 1
			if _, err := tx.ExecContext(ctx, `INSERT INTO room_sequences (room_id,next_seq) VALUES (?,?)`, roomID, current); err != nil {
				return err
			}
		default:
			return err
		}

		for i := range events {
			events[i].Seq = current + int64(i)
		}
		next := current + int64(len(events))
		if _, err := tx.ExecContext(ctx, `UPDATE room_sequences SET next_seq=? WHERE room_id=?`, next, roomID); err != nil {
			return err
		}

		for _, e := range events {
			if _, err := tx.ExecContext(ctx, `INSERT INTO events (room_id,seq,event_id,event_type,actor_user_id,causation_command_id,payload_json,server_ts) VALUES (?,?,?,?,?,?,?,?)`,
				e.RoomID, e.Seq, e.EventID, e.EventType, e.ActorUserID, e.CausationCommand, e.PayloadJSON, e.ServerTime); err != nil {
				return err
			}
		}

		if dedup != nil {
			if err := s.SaveDedupRecord(ctx, tx, *dedup); err != nil {
				return err
			}
		}
		if snap != nil {
			if err := s.SaveSnapshot(ctx, tx, *snap); err != nil {
				return err
			}
		}
		return nil
	})
}

func EncodeResultJSON(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
