package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"

	"github.com/pechorka/uuid"
)

type ValueNames struct {
	Name          []string
	Link          []string
	Theme         []string
	Sphere        []string
	KnowledgeType []string
	Subtype       []string
	Duration      []string
	WordCount     []string
}

var names = ValueNames{
	Name:          []string{"Название", "Name"},
	Link:          []string{"Ссылка", "Link"},
	Theme:         []string{"Тема", "Theme", "Topic"},
	Sphere:        []string{"Сфера", "#", "Sphere"},
	KnowledgeType: []string{"Тип", "Type"},
	Subtype:       []string{"Подтип", "Subtype"},
	Duration:      []string{"Длительность", "Duration"},
	WordCount:     []string{"Количество слов", "Word Count", "Word", "Слов", "Words", "Слова", "Слово"},
}

type texts struct {
	DefaultErrorText          string `json:"default_error_text"`
	NoLinkErrorText           string `json:"no_link_error_text"`
	InvalidDurationErrorText  string `json:"invalid_duration_error_text"`
	InvalidWordCountErrorText string `json:"invalid_wordcount_error_text"`
}

type knowledge struct {
	id    string
	name  string
	adder string
	// timeAdded     time.Time
	knowledgeType string //type - keyword in Go, so couldn't use it
	subtype       string
	theme         string
	sphere        string
	link          string
	wordCount     int
	duration      int
	//language      string
	// deleted       bool
	//notes 	string
	//file
	//tags []string
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

		//@pechor, Это нормально, что мы создаём пользователя по сути до старта..?
		userID := uuid.IntToUUID(msg.From.ID)
		userExists, _err := b.s.IsExists(userID)
		if _err == nil {
			if !userExists {
				createUser(b, msg)
			}
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
	//@pechor, раз мы запихнули создание пользователя в Run(), то нам тогда тут не надо его создавать, верно? и вот это всё надо бы удалить =>

	// log.Printf("[%s] %s", msg.From.UserName, msg.Text)

	// userID := uuid.IntToUUID(msg.From.ID)
	// userTgID := msg.From.ID
	// userTgName := msg.From.UserName
	// userTgFirstName := msg.From.FirstName
	// userTgLastName := msg.From.LastName
	// userTgLang := msg.From.LanguageCode

	// log.Printf("user id %q, tgName %q, name %q %q, lang %q", userID, userTgName, userTgFirstName, userTgLastName, userTgLang)

	// err := b.s.CreateUser(context.TODO(), userID, userTgID, userTgName, userTgFirstName, userTgLastName, userTgLang)
	// if err != nil {
	// 	log.Println("error while creating user", err)
	// 	b.reply(msg, b.t.DefaultErrorText)
	// }

	b.reply(msg, "Hello, human")
}

func (b *Bot) add(msg *tgbotapi.Message) {
	knw, err := b.parseKnowledge(msg.Text)
	if err != nil {
		log.Println("error while parsing knowledge", err)
		b.reply(msg, "failed to parse knowledge: "+err.Error())
		return
	}
	knw.adder = uuid.IntToUUID(msg.From.ID)

	addKnowledge(b, msg, knw)
	switch err {
	case nil:
		b.reply(msg, "thing added")
	default:
		b.reply(msg, "failed to add thing: "+err.Error())
	}
}

func (b *Bot) parseKnowledge(text string) (knowledge, error) { //method creating struct KNOWLEDGE from user input
	text = strings.TrimSpace(strings.TrimPrefix(text, "/add"))
	// text := msg.CommandArguments() - для команд, которые настоящие команды, а не которые пустую команду берут
	var err error
	var knw knowledge = knowledge{}

	split := strings.Split(text, "\n")
	for _, s := range split {
		s = strings.TrimSpace(s)
		if ContainsAny(s, "http://", "https://", "www.") || ContainsAny(s, names.Link...) {
			a := trimMeta(names.Link, s)
			if !strings.Contains(a, " ") {
				knw.link = a
			} else {
				return knowledge{}, errors.New(b.t.NoLinkErrorText) //TODO: подумать, может быть можно добавлять материалы без ссылок..?
			}
		}
		if ContainsAny(s, names.Name...) {
			knw.name = trimMeta(names.Name, s)
		}
		if ContainsAny(s, names.Theme...) {
			knw.theme = trimMeta(names.Theme, s)
		}
		if ContainsAny(s, names.Sphere...) {
			knw.sphere = trimMeta(names.Sphere, s)
		}
		if ContainsAny(s, names.KnowledgeType...) {
			knw.knowledgeType = trimMeta(names.KnowledgeType, s)
		}
		if ContainsAny(s, names.Subtype...) {
			knw.subtype = trimMeta(names.Subtype, s)
		}
		if ContainsAny(s, names.Duration...) {
			a := trimMeta(names.Duration, s)
			knw.duration, err = strconv.Atoi(a)
			if err != nil {
				log.Println("parsing error: ", err, "full line", s)
				return knowledge{}, errors.New(b.t.InvalidDurationErrorText) //TODO не падать, а закидывать нераспарсенное в заметки.
			}
		}
		if ContainsAny(s, names.WordCount...) {
			a := trimMeta(names.WordCount, s)
			knw.wordCount, err = strconv.Atoi(a)
			if err != nil {
				return knowledge{}, errors.New(b.t.InvalidWordCountErrorText) //TODO не падать, а закидывать нераспарсенное в заметки.
			}
		}

	}

	return knw, err
}

func ContainsAny(in string, contains ...string) bool { // function to check if there is something from array of strings in the beggining or end of text
	f := func(containsAllCase ...string) bool {
		for _, c := range containsAllCase {
			if strings.HasPrefix(in, c) || strings.HasSuffix(in, c) {
				return true
			}
		}
		return false
	}

	for _, c := range contains {
		contains = append(contains, strings.ToLower(c), strings.ToUpper(c))
	}

	return f(contains...)
}

func trimMeta(name []string, text string) (result string) { // method to delete meta information from line, such as "Name: XXXX" or "Name :XXX" or "XXXX - Name"
	result = text
	for _, s := range name {
		name = append(name, strings.ToLower(s), strings.ToUpper(s))
	}
	for i, s := range name {
		if strings.Contains(text, s) {
			fmt.Println(i, text, s)
			//TODO - проверить, что то, что написал Сергей - не хуйня
			// _, result, _ = strings.Cut(result, ":")
			// if index := strings.LastIndex(result, "-"); index > 0 {
			// 	result = result[:index]
			// }
			// result = strings.TrimSpace(result)
			result = strings.TrimSpace(result)
			result = strings.TrimPrefix(result, s)
			result = strings.TrimPrefix(result, ": ")
			result = strings.TrimPrefix(result, " :")
			result = strings.TrimPrefix(result, ":")
			result = strings.TrimSuffix(result, s)
			result = strings.TrimSuffix(result, " - ")
			result = strings.TrimSuffix(result, "- ")
			result = strings.TrimSuffix(result, " -")
			result = strings.TrimSpace(result)
		}
	}
	//TODO Убрать кавычки в оставшемся результате, см.  "case 3.1" в тестах функции
	return result
}

func addKnowledge(b *Bot, msg *tgbotapi.Message, knw knowledge) {
	// log.Printf("user id %q, link %q", userID, link)
	err := b.s.CreateKnowledge(context.TODO(), knw)
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
