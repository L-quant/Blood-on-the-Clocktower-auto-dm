package store

import (
	"context"
	"database/sql"
)

func (s *Store) CreateRoom(ctx context.Context, r Room) error {
	if s.MemoryMode {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.rooms[r.ID] = r
		return nil
	}
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO rooms (id,created_by,dm_user_id,status,created_at) VALUES (?,?,?,?,?)`,
		r.ID, r.CreatedBy, r.DMUserID, r.Status, r.CreatedAt,
	)
	if err != nil {
		return err
	}
	_, _ = s.DB.ExecContext(ctx, `INSERT INTO room_sequences (room_id,next_seq) VALUES (?,1) ON DUPLICATE KEY UPDATE next_seq=next_seq`, r.ID)
	return nil
}

func (s *Store) GetRoom(ctx context.Context, id string) (*Room, error) {
	if s.MemoryMode {
		s.mu.RLock()
		defer s.mu.RUnlock()
		if r, exists := s.rooms[id]; exists {
			return &r, nil
		}
		return nil, sql.ErrNoRows
	}
	row := s.DB.QueryRowContext(ctx, `SELECT id,created_by,dm_user_id,status,created_at FROM rooms WHERE id=?`, id)
	var r Room
	if err := row.Scan(&r.ID, &r.CreatedBy, &r.DMUserID, &r.Status, &r.CreatedAt); err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) AddRoomMember(ctx context.Context, m RoomMember) error {
	if s.MemoryMode {
		s.mu.Lock()
		defer s.mu.Unlock()
		members := s.members[m.RoomID]
		// Dedup
		for i, mem := range members {
			if mem.UserID == m.UserID {
				members[i] = m // update
				return nil
			}
		}
		s.members[m.RoomID] = append(members, m)
		return nil
	}
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO room_members (room_id,user_id,role,joined_at) VALUES (?,?,?,?) ON DUPLICATE KEY UPDATE role=VALUES(role)`,
		m.RoomID, m.UserID, m.Role, m.Joined,
	)
	return err
}

func (s *Store) GetRoomMembers(ctx context.Context, roomID string) ([]RoomMember, error) {
	if s.MemoryMode {
		s.mu.RLock()
		defer s.mu.RUnlock()
		return s.members[roomID], nil
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT room_id,user_id,role,joined_at FROM room_members WHERE room_id=?`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []RoomMember
	for rows.Next() {
		var m RoomMember
		if err := rows.Scan(&m.RoomID, &m.UserID, &m.Role, &m.Joined); err != nil {
			return nil, err
		}
		res = append(res, m)
	}
	return res, rows.Err()
}

func (s *Store) IsMember(ctx context.Context, roomID, userID string) (bool, string, error) {
	if s.MemoryMode {
		s.mu.RLock()
		defer s.mu.RUnlock()
		for _, m := range s.members[roomID] {
			if m.UserID == userID {
				return true, m.Role, nil
			}
		}
		return false, "", nil
	}
	row := s.DB.QueryRowContext(ctx, `SELECT role FROM room_members WHERE room_id=? AND user_id=?`, roomID, userID)
	var role string
	err := row.Scan(&role)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, "", nil
		}
		return false, "", err
	}
	return true, role, nil
}
