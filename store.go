package main

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type Store struct {
	db *sqlx.DB
}

func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

func (s *Store) CreateUser(ctx context.Context, id string, tg_id int64, tg_username string, tg_first_name string, tg_last_name string, tg_language string) error {
	_, err := s.db.ExecContext(ctx, "INSERT OR IGNORE INTO users(id, tg_id, tg_username, tg_first_name, tg_last_name, tg_language) VALUES ($1, $2, $3, $4, $5, $6)", id, tg_id, tg_username, tg_first_name, tg_last_name, tg_language)

	return errors.Wrap(err, "Creating user in db")
}

func (s *Store) IsExists(id string) (bool, error) {
	var e bool
	err := s.db.QueryRow("select 1 from users where id = $1", id).Scan(&e)
	return e, errors.Wrap(err, "seeking user")
}

func (s *Store) CreateKnowledge(ctx context.Context, id string, userID string, link string) error {
	_, err := s.db.ExecContext(ctx, "INSERT OR IGNORE INTO knowledge(id, adder, link, timeAdded) VALUES ($1, $2, $3, $4)", id, userID, link, time.Now())
	return errors.Wrap(err, "adding material")
}

func (s *Store) CreateKnowledgeFull(ctx context.Context, knowledge knowledge) error {
	_, err := s.db.ExecContext(ctx, "INSERT OR IGNORE INTO knowledge(id, adder, link, timeAdded) VALUES ($1, $2, $3, $4)", knowledge.id, knowledge.adder, knowledge.link, time.Now(), knowledge.knowledgeType, knowledge.subtype, knowledge.theme, knowledge.sphere, knowledge.wordCount, knowledge.duration, knowledge.language)
	return errors.Wrap(err, "adding material")
}

//@pechor, или лучше сделать одну функцию CreateKnowledge, которая принимает структуру и просто всё пихает в базу. Если будут null, so be it
