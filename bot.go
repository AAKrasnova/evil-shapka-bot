package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/pechorka/uuid"
)

type texts struct {
	DefaultErrorText string `json:"default_error_text"`
}

func readCMS(path string) (*texts, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	var c texts
	if err := json.NewDecoder(f).Decode(&c); err != nil {
		return nil, err
	}

	return &c, nil
}

type Bot struct {
	s   *Store
	bot *tgbotapi.BotAPI
}

func NewBot(s *Store, token string) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	return &Bot{s: s, bot: bot}, nil
}

func (b *Bot) Run() error {
	cms, err := readCMS("./cms.json")
	if err != nil {
		return err
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.bot.GetUpdatesChan(u)
	for update := range updates {
		msg := update.Message
		if msg == nil {
			continue
		}

		switch msg.Command() {
		case "add":
			b.add(msg, cms)
		case "start":
			b.start(msg, cms)
		case "":
			b.reply(msg, msg.Text)
		}
	}
	return nil
}

func (b *Bot) start(msg *tgbotapi.Message, cms *texts) {
	log.Printf("[%s] %s", msg.From.UserName, msg.Text)

	userID := uuid.IntToUUID(int(msg.From.ID))
	userTgName := msg.From.UserName
	userTgFirstName := msg.From.FirstName
	userTgLastName := msg.From.LastName
	userTgLang := msg.From.LanguageCode

	log.Printf("user id %q, tgName %q, name %q %q, lang %q", userID, userTgName, userTgFirstName, userTgLastName, userTgLang)

	err := b.s.CreateUser(context.TODO(), userID, userTgName, userTgFirstName, userTgLastName, userTgLang)
	if err != nil {
		log.Println("error while creating user", err)
		b.reply(msg, cms.DefaultErrorText)
	}
}

func (mngr *Bot) add(msg *tgbotapi.Message, cms *texts) {
	// text := strings.TrimSpace(strings.TrimPrefix(msg.Text, "/add "))

	// task, day, err := parseTaskAndDay(text)
	// if err != nil {
	// 	mngr.reply(msg, "failed to parse task: "+err.Error())
	// 	return
	// }

	// userID := uuid.IntToUUID(int(msg.From.ID))

	// err = mngr.secretary.AddTask(userID, task, day)
	// switch err {
	// case nil:
	// 	mngr.reply(msg, "task added")
	// default:
	// 	mngr.reply(msg, "failed to add task: "+err.Error())
	// }
}

func (b *Bot) reply(to *tgbotapi.Message, text string) {
	msg := tgbotapi.NewMessage(to.Chat.ID, text)
	msg.ReplyToMessageID = to.MessageID

	_, err := b.bot.Send(msg)
	if err != nil {
		log.Println("error while sending message: ", err)
	}
}
