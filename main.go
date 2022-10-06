package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"

	"github.com/jmoiron/sqlx"

	_ "github.com/mattn/go-sqlite3"
)

type config struct {
	TgToken string `json:"tg_token"`
	Local   bool   `json:"local"`
}

func readCfg(path string) (*config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	var c config
	if err := json.NewDecoder(f).Decode(&c); err != nil {
		return nil, err
	}

	return &c, nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := readCfg("./cfg.json")
	if err != nil {
		return err
	}

	if !cfg.Local {
		logFile, err := os.OpenFile("logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.ModePerm)
		if err != nil {
			return err
		}
		defer logFile.Close()

		log.SetOutput(logFile)
	}

	rawDB, err := sql.Open("sqlite3", "reminder.db")
	if err != nil {
		return err
	}

	return runBot(rawDB, cfg.TgToken)
}

func runBot(rawDB *sql.DB, token string) error {
	db := sqlx.NewDb(rawDB, "sqlite3")
	s := NewStore(db)
	b, err := NewBot(s, token)
	if err != nil {
		return err
	}
	return b.Run()
}
