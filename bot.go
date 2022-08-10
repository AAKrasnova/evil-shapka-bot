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

	"github.com/pechorka/uuid"
)

type texts struct {
	DefaultErrorText string `json:"default_error_text"`
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
	language      string
	// deleted       bool
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
		knw, err := parseKnowledge(text)
		knw.adder = uuid.IntToUUID(msg.From.ID)

		if err != nil {
			log.Println("error while parsing knowledge", err)
			b.reply(msg, "failed to parse knowledge: "+err.Error())
			return
		}

		addKnowledgeFull(b, msg, knw)
		switch err {
		case nil:
			b.reply(msg, "task added")
		default:
			b.reply(msg, "failed to add task: "+err.Error())
		}
	}
}

func parseKnowledge(text string) (knowledge, error) {
	var err error
	var knw knowledge = knowledge{ //это чтобы мне было понятно. Пусть пока тут будет
		id:            uuid.New(), //@pechor, лучше это делать тут (тут делать тупо как-то), в функции addKnowledgeFull \ addKnowledgeFast или вообще в функции в Store????????????
		name:          "",
		knowledgeType: "",
		subtype:       "",
		theme:         "",
		sphere:        "",
		link:          "",
		wordCount:     0,
		duration:      0,
		language:      ""}

	split := strings.Split(text, "\n")
	for i, s := range split {
		fmt.Println(i, s) //иначе он пишет, что i не используется, а мне и не надо её использовать лул
		if strings.Contains(s, "http://") || strings.Contains(s, "https://") || strings.Contains(s, "www.") || strings.Contains(s, "Ссылка") || strings.Contains(s, "Link") {
			a, ok := trimMeta([]string{"Ссылка", "Link"}, s)
			if ok {
				knw.link = a
			} else {
				err.Error() //хз что это
			}
		}
		if strings.Contains(s, "Название") || strings.Contains(s, "Name") {
			a, ok := trimMeta([]string{"Название", "Name"}, s)
			if ok {
				knw.name = a
			} else {
				err.Error() //хз что это
			}
		}
		if strings.Contains(s, "Тема") || strings.Contains(s, "Theme") || strings.Contains(s, "Topic") {
			a, ok := trimMeta([]string{"Тема", "Theme", "Topic"}, s)
			if ok {
				knw.theme = a
			} else {
				err.Error() //хз что это
			}
		}
		if strings.Contains(s, "Сфера") || strings.Contains(s, "#") || strings.Contains(s, "Sphere") {
			a, ok := trimMeta([]string{"Тема", "Theme", "Topic"}, s)
			if ok {
				knw.sphere = a
			} else {
				err.Error() //хз что это
			}
		}
		if strings.Contains(s, "Тип") || strings.Contains(s, "Type") {
			a, ok := trimMeta([]string{"Тип", "Type"}, s)
			if ok {
				knw.knowledgeType = a
			} else {
				err.Error() //хз что это
			}
		}
		if strings.Contains(s, "Подтип") || strings.Contains(s, "Subtype") {
			a, ok := trimMeta([]string{"Подтип", "Subtype"}, s)
			if ok {
				knw.subtype = a
			} else {
				err.Error() //хз что это
			}
		}
		if strings.Contains(s, "Длительность") || strings.Contains(s, "Duration") {
			a, ok := trimMeta([]string{"Длительность", "Duration"}, s)
			if ok {
				knw.duration, err = strconv.Atoi(a)
			} else {
				err.Error() //хз что это
			}
		}
		if strings.Contains(s, "Количество слов") || strings.Contains(s, "Word Count") || strings.Contains(s, "Word") {
			a, ok := trimMeta([]string{"Количество слов", "Word"}, s)
			if ok {
				knw.wordCount, err = strconv.Atoi(a)
			} else {
				err.Error() //хз что это
			}
		}

	}

	return knw, err
}

func trimMeta(name []string, text string) (result string, ok bool) {
	result = text
	for i, s := range name {
		if strings.Contains(text, s) {
			fmt.Println(i, text, s)
			result = strings.Trim(result, s)
		}
	}
	result = strings.Trim(result, ": ")
	// fmt.Print(strings.Trim("¡¡¡Hello, Gophers!!!", "!¡"))
	// Output:
	// Hello, Gophers

	if !strings.Contains(result, " ") {
		ok = true
	}

	return result, ok
}

//func addKnowledgeFull(b *Bot, msg *tgbotapi.Message, sphere string, name string, type string, subtype string, theme string, link string, wordCount string, duration string, language string) {
func addKnowledgeFull(b *Bot, msg *tgbotapi.Message, knw knowledge) {
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
