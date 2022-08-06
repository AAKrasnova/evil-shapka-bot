package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"

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
	t   *texts
}

func NewBot(s *Store, token string) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	cms, err := readCMS("./cms.json")
	if err != nil {
		return nil, err
	}
	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	return &Bot{s: s, bot: bot, t: cms}, nil
}

func (b *Bot) Run() error {

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
			b.add(msg)
		case "start":
			b.start(msg)
		case "":
			b.add(msg)
		}
	}
	return nil
}

func (b *Bot) start(msg *tgbotapi.Message) {
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
		b.reply(msg, b.t.DefaultErrorText)
	}
}

func (b *Bot) add(msg *tgbotapi.Message) {
	text := strings.TrimSpace(strings.TrimPrefix(msg.Text, "/add "))

	/* Если пользователь прислал только ссылку */
	if strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://") {
		if strings.Contains(text, " ") || strings.Contains(text, "\n") {
			b.reply(msg, "What do you want to do by"+text+",human?")
		} else {
			addKnowledge(b, msg, text)
		}
	}

	// task, day, err := parseTaskAndDay(text)
	// if err != nil {
	// 	b.reply(msg, "failed to parse task: "+err.Error())
	// 	return
	// }

	// userID := uuid.IntToUUID(int(msg.From.ID))

	// err = b.secretary.AddTask(userID, task, day)
	// switch err {
	// case nil:
	// 	b.reply(msg, "task added")
	// default:
	// 	b.reply(msg, "failed to add task: "+err.Error())
	// }
}

func addKnowledge(b *Bot, msg *tgbotapi.Message, link string) {
	userID := uuid.IntToUUID(int(msg.From.ID))
	knowledgeID := uuid.New()
	log.Printf("user id %q, link %q", userID, link)

	err := b.s.CreateKnowledge(context.TODO(), knowledgeID, userID, link)
	if err != nil {
		log.Println("error while creating knowledge", err)
		b.reply(msg, b.t.DefaultErrorText)
	}
}

func (b *Bot) reply(to *tgbotapi.Message, text string) {
	msg := tgbotapi.NewMessage(to.Chat.ID, text)
	msg.ReplyToMessageID = to.MessageID

	_, err := b.bot.Send(msg)
	if err != nil {
		log.Println("error while sending message: ", err)
	}
}
