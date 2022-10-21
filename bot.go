package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"

	"github.com/pechorka/uuid"
)

/*==================
USEFUL VALUES
===================*/

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

/*==================
CMS
===================*/

type texts struct {
	DefaultErrorText          string `json:"default_error_text"`
	NoLinkErrorText           string `json:"no_link_error_text"`
	InvalidDurationErrorText  string `json:"invalid_duration_error_text"`
	InvalidWordCountErrorText string `json:"invalid_wordcount_error_text"`
	StartDialogue             string `json:"start_dialogue"`
	FailedToParseKnowledge    string `json:"failed_parse_knowledge"`
	SuccessfullyAdded         string `json:"successfully_added"`
	FailedAddingKnowledge     string `json:"failed_adding_knowlegde"`
	FailedCreatingUser        string `json:"failed_creating_user"`
	SearchFailed              string `json:"search_failed"`
	KnowledgeName             string `json:"knowledge_name"`
	KnowledgeLink             string `json:"knowledge_link"`
	KnowledgeDuration         string `json:"knowledge_duration"`
	KnowledgeWordCount        string `json:"knowledge_wordcount"`
	KnowledgeSphere           string `json:"knowledge_sphere"`
	KnowledgeTheme            string `json:"knowledge_theme"`
	KnowledgeType             string `json:"knowledge_type"`
	KnowledgeSubtype          string `json:"knowledge_subtype"`
	KnowledgeAdder            string `json:"knowledge_adder"`
	KnowledgeTimeAdded        string `json:"knowledge_timeadded"`
	Words                     string `json:"words"`
	Minutes                   string `json:"minutes"`
	DidntFindAnything         string `json:"didnt_find_anything"`
	FailedLookingConsumed     string `json:"failed_looking_consumed"`
}

type localies map[string]texts

type knowledge struct {
	ID            string    `db:"id"`
	Name          string    `db:"name"`
	Adder         string    `db:"adder"`
	TimeAdded     time.Time `db:"timeAdded"`
	KnowledgeType string    `db:"type"` //type - keyword in Go, so couldn't use it
	Subtype       string    `db:"subtype"`
	Theme         string    `db:"theme"`
	Sphere        string    `db:"sphere"`
	Link          string    `db:"link"`
	WordCount     int       `db:"word_count"`
	Duration      int       `db:"duration"`
	//language      string `db:"language"`
	// deleted       bool `db:"deleted"`
	//notes 	string
	//file
	//tags []string
}

type user struct {
	ID          string `db:"id"`
	TGID        int64  `db:"tg_id"`
	TGUsername  string `db:"tg_username"`
	TGFirstName string `db:"tg_first_name"`
	TGLastName  string `db:"tg_last_name"`
	TGLanguage  string `db:"tg_language"`
}

func readCMS(path string, cms any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}

	return json.NewDecoder(f).Decode(&cms)
}

/*==================
TELEGRAM BOT
===================*/

type Bot struct {
	s   *Store
	bot *tgbotapi.BotAPI
	t   localies
}

func NewBot(s *Store, token string) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	var cms localies
	err = readCMS("./cms.json", &cms)
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
		if msg := update.Message; msg != nil {
			b.handleMsg(msg)
		}

		if callback := update.CallbackQuery; callback != nil {
			b.handleCallback(callback)
		}
	}
	return nil
}

func (b *Bot) handleMsg(msg *tgbotapi.Message) {
	b.ensureUserExists(msg)

	switch msg.Command() {
	case "add", "", "Add":
		b.add(msg)
	case "start", "Start":
		b.start(msg)
	case "find", "Find":
		b.find(msg)
	case "list", "List":
		b.findList(msg)
	}
}

func (b *Bot) handleCallback(callback *tgbotapi.CallbackQuery) {
	b.ensureUserExists(callback.Message)

	switch {
	case strings.HasPrefix(callback.Data, "read"):
		knwID := strings.TrimPrefix(callback.Data, "read")
		b.s.markAsRead(knwID, uuid.IntToUUID(callback.Message.From.ID))
	case strings.HasPrefix(callback.Data, "unread"):
		knwID := strings.TrimPrefix(callback.Data, "unread")
		b.s.markAsUnRead(knwID, uuid.IntToUUID(callback.Message.From.ID))
	}
}

/*
==================
TELEGRAM BOT: COMMANDS
===================
*/
func (b *Bot) start(msg *tgbotapi.Message) {
	b.replyWithText(msg, b.texts(msg).StartDialogue)
}

/*KNOWLEDGE MANAGEMENT*/
func (b *Bot) add(msg *tgbotapi.Message) {
	knw, err := b.parseKnowledge(msg)
	if err != nil {
		log.Println("error while parsing knowledge", err)
		b.replyWithText(msg, b.texts(msg).FailedToParseKnowledge+": "+err.Error())
		return
	}
	knw.Adder = uuid.IntToUUID(msg.From.ID)

	idKnowledge := ""
	idKnowledge, err = b.s.CreateKnowledge(knw)
	if err != nil {
		log.Println("error while creating knowledge", err.Error())
		b.replyWithText(msg, b.texts(msg).FailedAddingKnowledge)
	} else {
		knwldge, err1 := b.s.getKnowledgeById(idKnowledge)
		if err1 != nil {
			log.Println("error while retrieving created knowledge", err1)
			b.replyWithText(msg, b.texts(msg).FailedAddingKnowledge)
		} else {
			b.replyWithText(msg, b.texts(msg).SuccessfullyAdded+"\n"+b.FormatKnowledge(knwldge, false, msg))
		}
	}
}

func (b *Bot) parseKnowledge(msg *tgbotapi.Message) (knowledge, error) { //method creating struct KNOWLEDGE from user input
	text := strings.TrimSpace(strings.TrimPrefix(msg.Text, "/add"))
	// text := msg.CommandArguments() - для команд, которые настоящие команды, а не которые пустую команду берут
	var err error
	var knw knowledge = knowledge{}

	split := strings.Split(text, "\n")
	for _, s := range split {
		s = strings.TrimSpace(s)
		if ContainsAny(s, "http://", "https://", "www.") || ContainsAny(s, names.Link...) {
			a := trimMeta(names.Link, s)
			if !strings.Contains(a, " ") {
				knw.Link = a
			} else {
				return knowledge{}, errors.New(b.texts(msg).NoLinkErrorText) //TODO: подумать, может быть можно добавлять материалы без ссылок..?
			}
		}
		if ContainsAny(s, names.Name...) {
			knw.Name = trimMeta(names.Name, s)
		}
		if ContainsAny(s, names.Theme...) {
			knw.Theme = trimMeta(names.Theme, s)
		}
		if ContainsAny(s, names.Sphere...) {
			knw.Sphere = trimMeta(names.Sphere, s)
		}
		if ContainsAny(s, names.KnowledgeType...) {
			knw.KnowledgeType = trimMeta(names.KnowledgeType, s)
		}
		if ContainsAny(s, names.Subtype...) {
			knw.Subtype = trimMeta(names.Subtype, s)
		}
		if ContainsAny(s, names.Duration...) {
			a := trimMeta(names.Duration, s)
			knw.Duration, err = strconv.Atoi(a)
			if err != nil {
				log.Println("parsing error: ", err, "full line", s)
				return knowledge{}, errors.New(b.texts(msg).InvalidDurationErrorText) //TODO не падать, а закидывать нераспарсенное в заметки.
			}
		}
		if ContainsAny(s, names.WordCount...) {
			a := trimMeta(names.WordCount, s)
			knw.WordCount, err = strconv.Atoi(a)
			if err != nil {
				return knowledge{}, errors.New(b.texts(msg).InvalidWordCountErrorText) //TODO не падать, а закидывать нераспарсенное в заметки.
			}
		}

	}

	return knw, err
}

func (b *Bot) find(msg *tgbotapi.Message) {
	searchString := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(msg.Text, "/Find"), "/find"))
	userBDId := uuid.IntToUUID(msg.From.ID)
	consumed, err1 := b.s.getConsumedByUserId(userBDId)
	if err1 != nil {
		b.replyWithText(msg, b.texts(msg).FailedLookingConsumed+": "+err1.Error())
	}
	gotKnowledges, err := b.s.GetKnowledgeByUserAndSearch(userBDId, searchString)
	//TODO: <ANAL>: Сколько записей в среднем приходит? <H> Если пришло 100 записей, показать 3, а остальные показать по запросу
	if err == nil {
		if len(gotKnowledges) == 0 {
			b.replyWithText(msg, b.texts(msg).DidntFindAnything)
		} else {
			for _, knw := range gotKnowledges {
				btn := tgbotapi.NewInlineKeyboardButtonData("read", "read"+knw.ID)
				if consumed[knw.ID] {
					btn = tgbotapi.NewInlineKeyboardButtonData("unread", "unread"+knw.ID)
				}
				keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(btn))
				b.replyWithKeyboard(msg, b.FormatKnowledge(knw, true, msg), keyboard)
			}
		}
	} else {
		b.replyWithText(msg, b.texts(msg).SearchFailed+": "+err.Error())
	}

}
func (b *Bot) findList(msg *tgbotapi.Message) {
	searchString := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(msg.Text, "/FindList"), "/findList"), "/findlist"))
	userBDId := uuid.IntToUUID(msg.From.ID)
	gotKnowledges, err := b.s.GetKnowledgeByUserAndSearch(userBDId, searchString)
	//TODO: <ANAL>: Сколько записей в среднем приходит? <H> Если пришло 100 записей, показать 3, а остальные показать по запросу
	if err == nil {
		if len(gotKnowledges) == 0 {
			b.replyWithText(msg, b.texts(msg).DidntFindAnything)
		} else {
			answermessage := ""
			for _, knw := range gotKnowledges {
				answermessage += "\n" + b.FormatKnowledge(knw, false, msg)
			}
			b.replyWithText(msg, answermessage)
		}
	} else {
		b.replyWithText(msg, b.texts(msg).SearchFailed+": "+err.Error())
	}

}

/*==================
MAJOR SUPPORTING FUNCTIONS
===================*/

func (b *Bot) ensureUserExists(msg *tgbotapi.Message) {
	usr := user{
		ID:          uuid.IntToUUID(msg.From.ID),
		TGID:        msg.From.ID,
		TGUsername:  msg.From.UserName,
		TGFirstName: msg.From.FirstName,
		TGLastName:  msg.From.LastName,
		TGLanguage:  msg.From.LanguageCode,
	}
	_, err := b.s.CreateUser(usr)
	if err != nil {
		log.Println("error while creating user", err)
		b.replyWithText(msg, b.texts(msg).FailedCreatingUser)
	}
}

func (b *Bot) replyWithText(to *tgbotapi.Message, text string) {
	msg := tgbotapi.NewMessage(to.Chat.ID, text)
	msg.ReplyToMessageID = to.MessageID
	msg.ReplyMarkup = tgbotapi.ModeMarkdownV2
	b.send(msg)
}

func (b *Bot) replyWithKeyboard(to *tgbotapi.Message, text string, keyboard tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewMessage(to.Chat.ID, text)
	msg.ReplyToMessageID = to.MessageID
	msg.ReplyMarkup = keyboard
	msg.ReplyMarkup = tgbotapi.ModeMarkdownV2
	b.send(msg)
}

func (b *Bot) send(msg tgbotapi.MessageConfig) {
	_, err := b.bot.Send(msg)
	if err != nil {
		log.Println("error while sending message: ", err)
	}
}

/*==================
LITTLE HELPER FUNCTIONS
===================*/
/*KNOWLEDGE MANAGEMENT*/
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

func getLanguageCode(msg *tgbotapi.Message) string {
	lang := "en"
	if msg.From.LanguageCode == "ru" {
		lang = "ru"
	}
	return lang
}

func (b *Bot) texts(msg *tgbotapi.Message) texts {
	return b.t[getLanguageCode(msg)]
}

func (b *Bot) FormatKnowledge(knowledge knowledge, full bool, msg *tgbotapi.Message) string {
	str := ""
	if full {
		if len(knowledge.Name) > 0 {
			str += b.texts(msg).KnowledgeName + ": " + knowledge.Name
		}
		if len(knowledge.Link) > 0 {
			str += "\n" + b.texts(msg).KnowledgeLink + ": " + knowledge.Link
		}
		if len(knowledge.Sphere) > 0 {
			str += "\n" + b.texts(msg).KnowledgeSphere + ": " + knowledge.Sphere
		}
		if len(knowledge.KnowledgeType) > 0 {
			str += "\n" + b.texts(msg).KnowledgeType + ": " + knowledge.KnowledgeType
		}
		if len(knowledge.Subtype) > 0 {
			str += "\n" + b.texts(msg).KnowledgeSubtype + ": " + knowledge.Subtype
		}
		if len(knowledge.Theme) > 0 {
			str += "\n" + b.texts(msg).KnowledgeTheme + ": " + knowledge.Theme
		}
		if getLanguageCode(msg) == "ru" {
			str += "\n" + b.texts(msg).KnowledgeTimeAdded + ": " + knowledge.TimeAdded.Format("02.01.2006 15:04")
		} else {
			str += "\n" + b.texts(msg).KnowledgeTimeAdded + ": " + knowledge.TimeAdded.Format("Mon, 02 Jan 2006 03:04")
		}
		if knowledge.Duration > 0 {
			str += "\n" + b.texts(msg).KnowledgeDuration + ": " + strconv.Itoa(knowledge.Duration) + " " + b.texts(msg).Minutes
		}
		if knowledge.WordCount > 0 {
			str += "\n" + b.texts(msg).KnowledgeWordCount + ": " + strconv.Itoa(knowledge.WordCount) + " " + b.texts(msg).Words
		}
		//str += "\n" + b.texts(msg).KnowledgeAdder + ": " + knowledge.Adder //TODO: <H> Сделать красивое выведение имени, а не id пользователя 😆

	} else {
		//TODO сделать название кликабельным, а не отдельной строкой @pechorka, пока не поняла, как вообще добавлять Markup
		if len(knowledge.Name) > 0 {
			str += knowledge.Name
		}
		if len(knowledge.Link) > 0 {
			str += "\n" + knowledge.Link
		}
	}
	return str
}
