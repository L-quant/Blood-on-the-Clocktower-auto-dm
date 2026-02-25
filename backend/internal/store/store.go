// Package store 数据库连接与事务管理
//
// [OUT] api（数据库操作入口）
// [OUT] realtime（事件查询）
// [OUT] room（事件持久化）
// [POS] 数据访问层核心，提供 MySQL 连接池和事务封装
package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-sql-driver/mysql"
)

type Store struct {
	DB *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{DB: db}
}

func ConnectMySQL(dsn string) (*sql.DB, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}

func (s *Store) WithTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()
	if err := fn(tx); err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	tx = nil
	return nil
}

func (s *Store) Close() error { return s.DB.Close() }
