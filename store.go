package main

import (
	"log"
	"math/rand"
	"strings"
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
EVENT MANAGEMENT
===================*/
func (s *Store) CreateEvent(event event) (string, string, error) {
	idForCreating := uuid.New()
	event.Code = strings.Trim(event.Name, " ") + uuid.New()[:7]
	log.Println(idForCreating)
	_, err := s.db.Exec("INSERT INTO events(id, adder, name, timeAdded, code) VALUES ($1, $2, $3, $4, $5)",
		idForCreating, event.Adder, event.Name, time.Now(), event.Code)
	return idForCreating, event.Code, errors.Wrap(err, "adding event to db")
}

// func (s *Store) geteventById(id string) (event, error) {
// 	knw := event{}
// 	err := s.db.Get(&knw, "SELECT id, adder, link, name, timeAdded, type, subtype, theme, sphere, word_count, duration FROM event WHERE id=$1", id)
// 	//TODO: someday make SELECT *
// 	return knw, err
// }

// func (s *Store) GeteventByUserAndSearch(userID string, searchString string) ([]event, error) {
// 	knw := []event{}
// 	err := s.db.Select(&knw, "SELECT id, adder, link, name, timeAdded, type, subtype, theme, sphere, word_count, duration FROM event WHERE adder=$1 AND (name LIKE $2 OR link LIKE $2 OR sphere LIKE $2 OR type LIKE $2 OR subtype LIKE $2 OR theme LIKE $2)", userID, "%"+searchString+"%")
// 	//TODO: <QoL> make case insensitive
// 	//TODO: someday make SELECT *
// 	return knw, err
// }

func (s *Store) GetEventIDByCode(code string) (string, error) {
	ev := event{}
	err := s.db.Get(&ev, "SELECT id FROM events WHERE code=$1", code)
	return ev.ID, errors.Wrap(err, "searching event to by code")
}

/*==================
ENTRIES MANAGEMENT
===================*/
func (s *Store) CreateEntry(entry entry) (string, error) {
	idForCreating := uuid.New()
	entry.EventID, _ = s.GetEventIDByCode(entry.EventCode)
	//todo: do something with errors
	log.Println(idForCreating, entry.EventID)
	_, err := s.db.Exec("INSERT INTO entries(id, user_id, event_id, entry, timeAdded, drawn) VALUES ($1, $2, $3, $4, $5, $6)",
		idForCreating, entry.Adder, entry.EventID, entry.Entry, time.Now(), 9)
	return idForCreating, errors.Wrap(err, "adding entry to db")
}

func (s *Store) Draw(eventCode string) (entry, error) {
	eventID, _ := s.GetEventIDByCode(eventCode)
	entrs := []entry{}
	err := s.db.Get(&entrs, "SELECT id, event_id, user_id, entry, timeAdded, drawn FROM entries WHERE event_id=$1 AND drawn=9", eventID)
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	theEntry := entrs[(r1.Intn(len(entrs)))]
	//todo: не получилось присвоить - err
	s.db.Exec("UPDATE entries SET drawn=1 WHERE id=$1", theEntry.ID)
	theEntry.Drawn = 1

	return theEntry, errors.Wrap(err, "drawing")
}
