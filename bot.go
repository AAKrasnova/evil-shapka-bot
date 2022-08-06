package main

import (
	"context"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/pechorka/uuid"
)

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
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.bot.GetUpdatesChan(u)
	for update := range updates {
		if msg := update.Message; msg != nil { // If we got a message
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
				continue
			}

			switch msg.Command() {
			case "get":
				ok, err := b.s.IsExists(userID)
				if err != nil {
					log.Println(err)
					continue
				}
				if ok {
					b.reply(msg, "user exists")
				} else {
					b.reply(msg, "user don't exists")
				}
			}

		}
	}

	return nil
}

func (b *Bot) reply(to *tgbotapi.Message, text string) {
	msg := tgbotapi.NewMessage(to.Chat.ID, text)
	msg.ReplyToMessageID = to.MessageID

	_, err := b.bot.Send(msg)
	if err != nil {
		log.Println("error while sending message: ", err)
	}
}
