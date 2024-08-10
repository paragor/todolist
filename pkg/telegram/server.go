package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/paragor/todo/pkg/models"
	tele "gopkg.in/telebot.v3"
	"gopkg.in/telebot.v3/middleware"
	"log"
	"net/http"
	"time"
)

type TelegramServer struct {
	token           string
	userId          int64
	db              models.Repository
	serverPublicUrl string

	bot  *tele.Bot
	chat *tele.Chat

	cancel func()
}

func NewTelegramServer(token string, userId int64, serverPublicUrl string, db models.Repository) *TelegramServer {
	telegramServer := &TelegramServer{token: token, userId: userId, db: db, serverPublicUrl: serverPublicUrl}
	return telegramServer
}

func (t *TelegramServer) Start(ctx context.Context, stopper chan<- error) error {
	ctx, cancel := context.WithCancel(ctx)
	t.cancel = cancel
	b, err := tele.NewBot(tele.Settings{
		Token:  t.token,
		Poller: &tele.LongPoller{Timeout: 60 * time.Second},
		Client: &http.Client{
			Transport:     http.DefaultTransport,
			CheckRedirect: nil,
			Jar:           nil,
			Timeout:       time.Minute * 5,
		},
	})
	if err != nil {
		return fmt.Errorf("on init telegram bot: %w", err)
	}
	t.bot = b
	b.Use(middleware.Recover(func(err error, ctx tele.Context) {
		log.Printf("panic: %s\n", err.Error())
	}))
	b.Use(func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			data, err := json.Marshal(c.Update())
			if err == nil {
				log.Printf("telegram get msg: %s", string(data))
			}
			if c.Sender().ID != t.userId {
				log.Printf("telegram 403: %v", c.Message().Sender)
				return nil
			}
			_ = c.Notify(tele.Typing)
			if err := next(c); err != nil {
				_ = t.sendMessageHtml("error: " + err.Error())
				log.Printf("telegram ERROR: %s", err)
			}
			return err
		}
	})
	b.Handle("/start", func(c tele.Context) error {
		return t.sendMessageHtml(fmt.Sprintf(`<a href="%s">Welcome!</a>`, t.serverPublicUrl), t.withMainPageWebApp())
	})
	b.Handle("/agenda", func(c tele.Context) error {
		return t.TriggerAgenda()
	})
	b.Handle("/help", func(c tele.Context) error {
		return t.sendMessageHtml(models.HumanInputHelp)
	})
	b.Handle(tele.OnText, func(c tele.Context) error {
		return t.humanInput(c.Message().Text)
	})
	commands := []tele.Command{
		{
			Text:        "help",
			Description: "Help",
		},
		{
			Text:        "start",
			Description: "Web app link",
		},
		{
			Text:        "agenda",
			Description: "Show agenda",
		},
	}
	if err := b.SetCommands(commands); err != nil {
		return fmt.Errorf("cant set commands: %w", err)
	}

	t.chat, err = b.ChatByID(t.userId)
	if err != nil {
		return fmt.Errorf("cant get chat: %w", err)
	}
	go func() {
		t.bot.Start()
		stopper <- fmt.Errorf("stop telegram")
	}()

	notifier := newNotifier(t.db, t)
	go func() {
		err := notifier.Start(ctx)
		stopper <- fmt.Errorf("stop telegram.notifier: %w", err)
	}()
	go func() {
		<-ctx.Done()
		t.bot.Stop()
	}()

	return nil
}

func (t *TelegramServer) Stop() {
	if t.cancel != nil {
		t.cancel()
	}
}
