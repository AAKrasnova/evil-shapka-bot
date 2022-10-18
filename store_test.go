package main

import (
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestStore_CreateKnowledge(t *testing.T) {

	tests := []struct {
		name      string
		knowledge knowledge
	}{
		{
			name: "case 1",
			knowledge: knowledge{
				Name:          "Webinar: Importance of Market Research & Cognitive Design by Amazon Sr PM",
				Adder:         "",
				KnowledgeType: "Video",
				Subtype:       "Webinar",
				Theme:         "Market Research",
				Sphere:        "PM",
				Link:          "https://www.linkedin.com/video/event/urn:li:ugcPost:6950083329849221120/",
				WordCount:     0,
				Duration:      10,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, cleanup := prepDB(t)
			t.Cleanup(cleanup)
			str := NewStore(db)
			idCreated, err := str.CreateKnowledge(tt.knowledge)
			if err != nil {
				t.Errorf("Store.CreateKnowledge() error = %v", err)
			}
			gotKnw, err := str.getKnowledgeById(idCreated)
			if err != nil {
				t.Errorf("Store.getKnowledgeById() error = %v", err)
			}
			tt.knowledge.ID = idCreated
			tt.knowledge.TimeAdded = gotKnw.TimeAdded // don't care about this field
			require.Equal(t, tt.knowledge, gotKnw)
		})
	}
}

func TestStore_CreateUser(t *testing.T) {
	tests := []struct {
		name string
		user user
	}{
		{
			name: "case 1",
			user: user{
				ID:          "",
				TGID:        23232556,
				TGUsername:  "RubellaTest",
				TGFirstName: "Тест",
				TGLastName:  "Тестович",
				TGLanguage:  "ru",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, cleanup := prepDB(t)
			t.Cleanup(cleanup)
			str := NewStore(db)
			idCreated, err := str.CreateUser(tt.user)
			if err != nil {
				t.Errorf("Store.CreateUser() error = %v", err)
			}
			gotUser, err := str.getUserById(idCreated)
			if err != nil {
				t.Errorf("Store.getUserById() error = %v", err)
			}
			tt.user.ID = idCreated
			require.Equal(t, tt.user, gotUser)
		})
	}
}

func prepDB(t *testing.T) (db *sqlx.DB, cleanup func()) {
	t.Helper()

	dbPath := fmt.Sprintf("test%d.db", rand.Int31())
	cleanup = func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
		_ = os.Remove(dbPath)
	}
	db, err := sqlx.Connect("sqlite3", dbPath)
	if err != nil {
		cleanup()
		t.Fatal(err)
	}
	driver, err := sqlite3.WithInstance(db.DB, &sqlite3.Config{})
	if err != nil {
		cleanup()
		t.Fatal(err)
	}
	m, err := migrate.NewWithDatabaseInstance("file://migrations", "ql", driver)
	if err != nil {
		cleanup()
		t.Fatal(err)
	}
	if err := m.Up(); err != nil {
		cleanup()
		t.Fatal(err)
	}

	return db, cleanup
}
