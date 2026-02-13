package store

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
)

type Store struct {
	DB         *sql.DB
	MemoryMode bool
	mu         sync.RWMutex
	users      map[string]User
	rooms      map[string]Room
	members    map[string][]RoomMember
	events     map[string][]StoredEvent
	snapshots  map[string]Snapshot
	dedups     map[string]DedupRecord
}

func New(db *sql.DB) *Store {
	return &Store{DB: db}
}

func NewMemoryStore() *Store {
	return &Store{
		MemoryMode: true,
		users:      make(map[string]User),
		rooms:      make(map[string]Room),
		members:    make(map[string][]RoomMember),
		events:     make(map[string][]StoredEvent),
		snapshots:  make(map[string]Snapshot),
		dedups:     make(map[string]DedupRecord),
	}
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

	// Ping to verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}

func (s *Store) WithTx(ctx context.Context, fn func(*sql.Tx) error) error {
	if s.MemoryMode {
		return fn(nil) // Pass nil transaction, caller must handle nil if logic is shared
	}
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

func (s *Store) Close() error {
	if s.MemoryMode {
		return nil
	}
	return s.DB.Close()
}
