package main

import (
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

	"github.com/pechorka/uuid"
)

type Store struct {
	db *sqlx.DB
}

func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

/*==================
USER MANAGEMENT
===================*/

func (s *Store) CreateUser(user user) (string, error) {
	idForCreating := uuid.New()
	_, err := s.db.Exec("INSERT OR IGNORE INTO users(id, tg_id, tg_username, tg_first_name, tg_last_name, tg_language) VALUES ($1, $2, $3, $4, $5, $6)", idForCreating, user.TGID, user.TGUsername, user.TGFirstName, user.TGLastName, user.TGLanguage)

	return idForCreating, errors.Wrap(err, "Creating user in db")
}

func (s *Store) getUserById(id string) (user, error) {
	usr := user{}
	err := s.db.Get(&usr, "SELECT * FROM users WHERE id=$1", id)
	return usr, err
}

func (s *Store) GetUserByTelegramId(TGID string) (user, error) {
	usr := user{}
	err := s.db.Get(&usr, "SELECT * FROM users WHERE tg_id=$1", TGID)
	return usr, err
}

/*==================
KNOWLEDGE MANAGEMENT
===================*/

func (s *Store) CreateKnowledge(knowledge knowledge) (string, error) {
	idForCreating := uuid.New()
	log.Println(idForCreating)
	_, err := s.db.Exec("INSERT INTO knowledge(id, adder, link, name, timeAdded, type, subtype, theme, sphere, word_count, duration) VALUES ($1, $2, $3, $4, $5, $6, $7, $8,$9, $10, $11)",
		idForCreating, knowledge.Adder, knowledge.Link, knowledge.Name, time.Now(), knowledge.KnowledgeType, knowledge.Subtype, knowledge.Theme, knowledge.Sphere,
		knowledge.WordCount, knowledge.Duration)
	return idForCreating, errors.Wrap(err, "adding material to db")
}

func (s *Store) getKnowledgeById(id string) (knowledge, error) {
	knw := knowledge{}
	err := s.db.Get(&knw, "SELECT id, adder, link, name, timeAdded, type, subtype, theme, sphere, word_count, duration FROM knowledge WHERE id=$1", id)
	//TODO: someday make SELECT *
	return knw, err
}

func (s *Store) GetKnowledgeByUserAndSearch(userID string, searchString string) ([]knowledge, error) {
	knw := []knowledge{}
	err := s.db.Select(&knw, "SELECT id, adder, link, name, timeAdded, type, subtype, theme, sphere, word_count, duration FROM knowledge WHERE adder=$1 AND (name LIKE $2 OR link LIKE $2 OR sphere LIKE $2 OR type LIKE $2 OR subtype LIKE $2 OR theme LIKE $2)", userID, "%"+searchString+"%")
	//TODO: <QoL> make case insensitive
	//TODO: someday make SELECT *
	return knw, err
}
