package store

import (
	"context"
	"database/sql"
)

func (s *Store) CreateUser(ctx context.Context, u User) error {
	if s.MemoryMode {
		s.mu.Lock()
		defer s.mu.Unlock()
		if _, exists := s.users[u.ID]; exists {
			return nil // Already exists
		}
		s.users[u.ID] = u
		return nil
	}
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO users (id,email,password_hash,created_at) VALUES (?,?,?,?)`,
		u.ID, u.Email, u.PasswordHash, u.CreatedAt,
	)
	return err
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	if s.MemoryMode {
		s.mu.RLock()
		defer s.mu.RUnlock()
		for _, u := range s.users {
			if u.Email == email {
				return &u, nil
			}
		}
		return nil, sql.ErrNoRows
	}
	row := s.DB.QueryRowContext(ctx, `SELECT id,email,password_hash,created_at FROM users WHERE email=?`, email)
	var u User
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) GetUserByID(ctx context.Context, id string) (*User, error) {
	if s.MemoryMode {
		s.mu.RLock()
		defer s.mu.RUnlock()
		if u, exists := s.users[id]; exists {
			return &u, nil
		}
		return nil, sql.ErrNoRows
	}
	row := s.DB.QueryRowContext(ctx, `SELECT id,email,password_hash,created_at FROM users WHERE id=?`, id)
	var u User
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}
