package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"

	"github.com/pechorka/uuid"
)

/*==================
USEFUL VALUES
===================*/

const Udentified = 0
const Yes = 1
const No = 9

/*==================
CMS
===================*/

type texts struct {
	DefaultErrorText   string `json:"default_error_text"`
	StartDialogue      string `json:"start_dialogue"`
	FailedCreatingUser string `json:"failed_creating_user"`
	FailedCreatingEvent string `json:"failed_creating_event"`
	YourCode string `json:"your_code"`
	CopyByClicking string `json:"copy_by_clicking"`
	TryCreateEntry string `json:"try_create_entry"`
	FailedCreatingEntry string `json:"failed_creating_entry"`
	EntryAdded string `json:"entry_added"`
	FailedDrawingEntry string `json:"failed_drawing_entry"`
	YouDrewEntry string `json:"you_drew_entry"`
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

/*==================
TYPES
===================*/

type event struct {
	ID        string    `db:"id"`
	Name      string    `db:"name"`
	Adder     string    `db:"adder"`
	TimeAdded time.Time `db:"timeAdded"`
	Code      string    `db:"code"`
}

type user struct {
	ID          string `db:"id"`
	TGID        int64  `db:"tg_id"`
	TGUsername  string `db:"tg_username"`
	TGFirstName string `db:"tg_first_name"`
	TGLastName  string `db:"tg_last_name"`
	TGLanguage  string `db:"tg_language"`
}

type entry struct {
	ID        string    `db:"id"`
	EventID      string    `db:"event_id"`
	EventCode string
	Adder     string    `db:"user_id"`
	TimeAdded time.Time `db:"timeAdded"`
	Entry      string    `db:"entry"`
	Drawn int64  `db:"drawn"` //  Udentified = 0, Yes = 1,  No = 9
}

/*==================
TELEGRAM BOT
===================*/

type Bot struct {
	s     *Store
	bot   *tgbotapi.BotAPI
	cms   *localies
	debug bool
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
	bot.Debug = true // TODO before release take from config

	return &Bot{s: s, bot: bot, cms: cms, debug: true}, nil
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
	case "NewEvent", "newevent":
		b.newEvent(msg)
	case "Put", "put":
		b.put(msg)
	case "Draw", "draw":
		b.draw(msg)
	case "Start", "start":
		b.start(msg)
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

/*EVENT MANAGEMENT*/
func (b *Bot) newEvent(msg *tgbotapi.Message) {
	evt:=event {
		Name:  msg.CommandArguments(),
		Adder: uuid.IntToUUID(msg.From.ID),
	}

	_, evt.Code, err: = b.s.CreateEvent(evt)
	if err != nil {
		log.Println("error while creating event", err)
		b.replyWithText(msg, b.texts(msg).FailedCreatingEvent)
	}
	b.replyWithText(msg, b.texts(msg).YourCode+" `"+evt.Code+"` "+"b.texts(msg).CopyByClicking"+"b.texts(msg).TryCreateEntry")
}

func (b *Bot) put(msg *tgbotapi.Message) {
	codeentry := strings.SplitN(msg.CommandArguments(), ":", 2)
	// a := strings.SplitN("Code24141: Masha Ivanova", ":", 2)
	// fmt.Println(a[0])
	// fmt.Println(a[1])
//>>>> 	Code24141
//>>>>  Masha Ivanova

//TODO: parsing error
	entr:= entry{
		Adder: uuid.IntToUUID(msg.From.ID),
		EventCode: codeentry[0],
		Entry: strings.TrimSpace(codeentry[1])
	}

	_, err: = b.s.CreateEntry(entr)
	if err != nil {
		log.Println("error while creating entry", err)
		b.replyWithText(msg, b.texts(msg).FailedCreatingEntry)
	}
	b.replyWithText(msg, b.texts(msg).EntryAdded+" "+entr.ID)
}

func (b *Bot) draw(msg *tgbotapi.Message) {
	code:= msg.CommandArguments()
	theEntry, err:= b.s.Draw(code)
	
	if err != nil {
		log.Println("error while drawing entry", err)
		b.replyWithText(msg, b.texts(msg).FailedDrawingEntry)
	}
	b.replyWithText(msg, b.texts(msg).YouDrewEntry+" "+theEntry.Entry)

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
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	b.send(msg)
}

func (b *Bot) replyDebug(to *tgbotapi.Message, text string) {
	if !b.debug {
		return
	}
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
	b.send(msg)
}

func (b *Bot) replyWithKeyboard(to *tgbotapi.Message, text string, keyboard tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewMessage(to.Chat.ID, text)
	msg.ReplyToMessageID = to.MessageID
	msg.ReplyMarkup = keyboard
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
