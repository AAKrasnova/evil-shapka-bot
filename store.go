package main

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type Store struct {
	db *sqlx.DB
}

func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

func (s *Store) CreateUser(ctx context.Context, id string, tg_username string, tg_first_name string, tg_last_name string, tg_language string) error {
	_, err := s.db.ExecContext(ctx, "INSERT OR IGNORE INTO users(id, tg_username, tg_first_name, tg_last_name, tg_language) VALUES ($1, $2, $3, $4, $5)", id, tg_username, tg_first_name, tg_last_name, tg_language)

	return errors.Wrap(err, "Creating user in db")
}

func (s *Store) IsExists(id string) (bool, error) {
	var e bool
	err := s.db.QueryRow("select 1 from users where id = $1", id).Scan(&e)
	return e, errors.Wrap(err, "seeking user")
}
