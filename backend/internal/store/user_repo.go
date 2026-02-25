// Package store 用户账号 CRUD 操作
//
// [OUT] api（用户注册与登录查询）
// [POS] 用户存储层，处理用户创建与按邮箱/ID 查询
package store

import (
	"context"
)

func (s *Store) CreateUser(ctx context.Context, u User) error {
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO users (id,email,password_hash,created_at) VALUES (?,?,?,?)`,
		u.ID, u.Email, u.PasswordHash, u.CreatedAt,
	)
	return err
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT id,email,password_hash,created_at FROM users WHERE email=?`, email)
	var u User
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) GetUserByID(ctx context.Context, id string) (*User, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT id,email,password_hash,created_at FROM users WHERE id=?`, id)
	var u User
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}
