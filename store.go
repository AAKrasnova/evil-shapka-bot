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

func (s *Store) CreateUser(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "INSERT OR IGNORE INTO users(id) VALUES ($1)", id)
	return errors.Wrap(err, "creating user in db")
}

func (s *Store) IsExists(id string) (bool, error) {
	var e bool
	err := s.db.QueryRow("select 1 from users where id = $1", id).Scan(&e)
	return e, errors.Wrap(err, "seeking user")
}
