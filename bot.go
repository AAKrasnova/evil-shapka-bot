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

	userID := uuid.IntToUUID(msg.From.ID)
	userTgID := msg.From.ID
	userTgName := msg.From.UserName
	userTgFirstName := msg.From.FirstName
	userTgLastName := msg.From.LastName
	userTgLang := msg.From.LanguageCode

	log.Printf("user id %q, tgName %q, name %q %q, lang %q", userID, userTgName, userTgFirstName, userTgLastName, userTgLang)

	err := b.s.CreateUser(context.TODO(), userID, userTgID, userTgName, userTgFirstName, userTgLastName, userTgLang)
	if err != nil {
		log.Println("error while creating user", err)
		b.reply(msg, b.t.DefaultErrorText)
	}
}

func (b *Bot) add(msg *tgbotapi.Message) {
	text := strings.TrimSpace(strings.TrimPrefix(msg.Text, "/add "))
	// text := msg.CommandArguments() - для команд, которые настоящие команды, а не которые пустую команду берут

	/* Если пользователь прислал только ссылку */
	if strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://") {
		if strings.Contains(text, " ") || strings.Contains(text, "\n") { //Это типа проверка на то, чтобы была только ссылка и ничего больше
			b.reply(msg, "What do you want to do by "+text+", human?")
		} else {
			addKnowledgeFast(b, msg, text)
		}
	} else {
		args, err := parseKnowledge(text)
		if err != nil {
			log.Println("error while parsing knowledge", err)
			b.reply(msg, "failed to parse knowledge: "+err.Error())
			return
		}

		err = addKnowledgeFull(b, msg, args)
		switch err {
		case nil:
			b.reply(msg, "task added")
		default:
			b.reply(msg, "failed to add task: "+err.Error())
		}
	}
}

func (args []string) parseKnowledge(text string) {
	//TODO правильно ли я возвращаю массив? А как возвращать keyvalue?

}

//func addKnowledgeFull(b *Bot, msg *tgbotapi.Message, sphere string, name string, type string, subtype string, theme string, link string, wordCount string, duration string, language string) {
func addKnowledgeFull(b *Bot, msg *tgbotapi.Message, args []string) {
	// @pechor, где лучше преобразовывать юзерИД? В каждой функции addKnowledge или передавать уже в неё как аргумент?
}

func addKnowledgeFast(b *Bot, msg *tgbotapi.Message, link string) {
	userID := uuid.IntToUUID(msg.From.ID)
	userExists, _err := b.s.IsExists(userID)
	if _err == nil {
		if !userExists {
			createUser(b, msg)
		}
	}

	knowledgeID := uuid.New()
	log.Printf("user id %q, link %q", userID, link)

	err := b.s.CreateKnowledge(context.TODO(), knowledgeID, userID, link)
	if err != nil {
		log.Println("error while creating knowledge", err)
		b.reply(msg, b.t.DefaultErrorText)
	} else {
		b.reply(msg, "Успешно добавлено!")
	}
}

func createUser(b *Bot, msg *tgbotapi.Message) {
	userID := uuid.IntToUUID(msg.From.ID)

	userExists, _err := b.s.IsExists(userID)
	if _err == nil {
		if !userExists {
			userTgID := msg.From.ID
			userTgName := msg.From.UserName
			userTgFirstName := msg.From.FirstName
			userTgLastName := msg.From.LastName
			userTgLang := msg.From.LanguageCode
			err := b.s.CreateUser(context.TODO(), userID, userTgID, userTgName, userTgFirstName, userTgLastName, userTgLang)
			if err != nil {
				log.Println("error while creating user", err)
				b.reply(msg, b.t.DefaultErrorText)
			}
		}
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
