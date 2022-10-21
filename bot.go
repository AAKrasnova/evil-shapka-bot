package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"

	"github.com/gocarina/gocsv"
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

const Udentified = 0
const Yes = 1
const No = 9

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
	KnowledgeIsRead           string `json:"knowledge_is_read"`
	KnowledgeIsNotRead        string `json:"knowledge_is_not_read"`
	DidntFindAnything         string `json:"didnt_find_anything"`
	FailedLookingConsumed     string `json:"failed_looking_consumed"`
}

type localies struct {
	mu  sync.RWMutex
	cms map[string]texts

	watcher *fsnotify.Watcher
	stop    chan struct{}
}

func newLocalies() *localies {
	l := localies{
		cms: make(map[string]texts),
	}
	return &l
}

func (l *localies) initWatcher(path string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.watcher != nil { // already watching
		return nil
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return errors.Wrap(err, "failed to create watcher")
	}
	err = watcher.Add(path)
	if err != nil {
		return errors.Wrap(err, "failed to add file to watcher")
	}
	l.watcher = watcher
	l.stop = make(chan struct{})
	go func() {
		for {
			select {
			case event := <-l.watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {
					err := l.load(path)
					if err != nil {
						log.Println(errors.Wrap(err, "failed to reload cms"))
					}
				}
			case err := <-l.watcher.Errors:
				log.Println(errors.Wrap(err, "failed to watch cms"))
			case <-l.stop:
				l.clearWatcher()
				return
			}
		}
	}()
	return nil
}

func (l *localies) clearWatcher() {
	if l.watcher == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	err := l.watcher.Close()
	if err != nil {
		log.Println(errors.Wrap(err, "failed to close watcher"))
	}
	l.watcher = nil
	l.stop = nil
}

func (l *localies) close() {
	close(l.stop)
}

func (l *localies) load(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	var cms map[string]texts
	err = json.NewDecoder(f).Decode(&cms)
	if err != nil {
		return err
	}
	l.cms = cms
	return f.Close()
}

func (l *localies) texts(msg *tgbotapi.Message) texts {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.cms[getLanguageCode(msg)]
}

type knowledge struct {
	ID            string    `db:"id" csv:"-"`
	Name          string    `db:"name" csv:"Name"`
	Adder         string    `db:"adder" csv:"-"`
	TimeAdded     time.Time `db:"timeAdded" csv:"-"`
	KnowledgeType string    `db:"type" csv:"Type"` //type - keyword in Go, so couldn't use it
	Subtype       string    `db:"subtype" csv:"Subtype"`
	Theme         string    `db:"theme" csv:"Theme"`
	Sphere        string    `db:"sphere" csv:"Sphere"`
	Link          string    `db:"link" csv:"Link"`
	WordCount     int       `db:"word_count" csv:"Word Count"`
	Duration      int       `db:"duration" csv:"Duration"`
	//language      string `db:"language"`
	// deleted       bool `db:"deleted"`
	//file
	//tags []string
	isRead int `csv:"-"`
	//	DateConsumed time.Time `csv:"Date Consumed"`
	ReadyToRe int    `csv:"ReadyToRe"`
	Notes     string `csv:"Notes"`
}

type user struct {
	ID          string `db:"id"`
	TGID        int64  `db:"tg_id"`
	TGUsername  string `db:"tg_username"`
	TGFirstName string `db:"tg_first_name"`
	TGLastName  string `db:"tg_last_name"`
	TGLanguage  string `db:"tg_language"`
}

/*==================
TELEGRAM BOT
===================*/

type Bot struct {
	s   *Store
	bot *tgbotapi.BotAPI
	cms *localies
}

func NewBot(s *Store, token string) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	log.Printf("Authorized on account %s", bot.Self.UserName)

	cms := newLocalies()
	err = cms.load("./cms.json")
	if err != nil {
		return nil, err
	}
	err = cms.initWatcher("./cms.json")
	if err != nil {
		return nil, err
	}
	bot.Debug = true

	return &Bot{s: s, bot: bot, cms: cms}, nil
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

func (b *Bot) Stop() {
	b.bot.StopReceivingUpdates()
	b.cms.close()
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
	case "import", "Import":
		b.importKnowledges(msg)
	}
}

func (b *Bot) handleCallback(callback *tgbotapi.CallbackQuery) {
	// b.ensureUserExists(callback.Message) //@pechorka, TODO

	switch {
	case strings.HasPrefix(callback.Data, "read"):
		knwID := strings.TrimPrefix(callback.Data, "read")
		b.s.markAsRead(knwID, uuid.IntToUUID(callback.From.ID))
	case strings.HasPrefix(callback.Data, "unread"):
		knwID := strings.TrimPrefix(callback.Data, "unread")
		b.s.markAsUnRead(knwID, uuid.IntToUUID(callback.From.ID))
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
				// We dont use udentified (0) because we have map with only TRUE things. All others aer NO. And we know it
				knw.isRead = No
				if consumed[knw.ID] {
					knw.isRead = Yes
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
	searchString := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(msg.Text, "/List"), "/list"))
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

/*
==================
IMPORT AND EXPORT
===================
*/
func (b *Bot) importKnowledges(msg *tgbotapi.Message) {
	fileLink, err := b.bot.GetFileDirectURL(msg.ReplyToMessage.Document.FileID)
	if err != nil {
		b.replyError(msg, b.texts(msg).FailedToGetFile, errors.Wrap(err, "failed to get file link"))
		return
	}

	resp, err := http.Get(fileLink) // TODO file might be too malicious - check it somehow
	if err != nil {
		b.replyError(msg, b.texts(msg).FailedToGetFile, errors.Wrap(err, "failed to get file"))
		return
	}
	defer resp.Body.Close()

	knowledges, err := b.parseCSV(resp.Body)
	if err != nil {
		b.replyError(msg, b.texts(msg).FailedToParseFile, errors.Wrap(err, "failed to parse file"))
		return
	}

	// TODO save knowledges to DB
}

func (b *Bot) parseCSV(data io.Reader) ([]*knowledge, error) {
	knowledges := []*knowledge{}

	if err := gocsv.Unmarshal(data, &knowledges); err != nil { // Load things from file
		return nil, err
	}

	return knowledges, nil
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
	// msg.ReplyMarkup = tgbotapi.ModeMarkdownV2
	b.send(msg)
}

func (b *Bot) replyError(to *tgbotapi.Message, text string, err error) {
	msg := tgbotapi.NewMessage(to.Chat.ID, text)
	msg.ReplyToMessageID = to.MessageID
	if err != nil {
		log.Println(err.Error())
	}
	// msg.ReplyMarkup = tgbotapi.ModeMarkdownV2
	b.send(msg)
}

func (b *Bot) replyWithKeyboard(to *tgbotapi.Message, text string, keyboard tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewMessage(to.Chat.ID, text)
	msg.ReplyToMessageID = to.MessageID
	msg.ReplyMarkup = keyboard
	// msg.ReplyMarkup = tgbotapi.ModeMarkdownV2
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
	return b.cms.texts(msg)
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
		if knowledge.isRead != Udentified {
			if knowledge.isRead == Yes {
				str += "\n" + b.texts(msg).KnowledgeIsRead
			}
			if knowledge.isRead == No {
				str += "\n" + b.texts(msg).KnowledgeIsNotRead
			}

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
