package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
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
	commands := []tele.Command{
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

	notifier := newNotifier(time.Second*63, t.db, t)
	go func() {
		err := notifier.start(ctx)
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

type sendOption func(o *tele.SendOptions)

func (t *TelegramServer) withMainPageWebApp() sendOption {
	return func(o *tele.SendOptions) {
		reply := &tele.ReplyMarkup{}
		reply.Inline(
			reply.Split(
				1,
				[]tele.Btn{
					reply.WebApp(
						"Main page",
						&tele.WebApp{URL: t.serverPublicUrl + "/"},
					),
				},
			)...,
		)
		o.ReplyMarkup = reply
	}
}
func (t *TelegramServer) withAgendaWebApp() sendOption {
	return func(o *tele.SendOptions) {
		reply := &tele.ReplyMarkup{}
		reply.Inline(
			reply.Split(
				1,
				[]tele.Btn{
					reply.WebApp(
						"Agenda",
						&tele.WebApp{URL: t.serverPublicUrl + "/agenda"},
					),
				},
			)...,
		)
		o.ReplyMarkup = reply
	}
}
func (t *TelegramServer) withTaskWebApp(uuid uuid.UUID) sendOption {
	return func(o *tele.SendOptions) {
		reply := &tele.ReplyMarkup{}
		reply.Inline(
			reply.Split(
				1,
				[]tele.Btn{
					reply.WebApp(
						"Task info",
						&tele.WebApp{URL: t.serverPublicUrl + "/task?uuid=" + uuid.String()},
					),
				},
			)...,
		)
		o.ReplyMarkup = reply
	}
}

func (t *TelegramServer) withEnableNotifications() sendOption {
	return func(o *tele.SendOptions) {
		o.DisableNotification = false
	}
}

func (t *TelegramServer) sendMessageHtml(msg string, options ...sendOption) error {
	sendOptions := &tele.SendOptions{
		DisableNotification:   true,
		DisableWebPagePreview: true,
		ParseMode:             tele.ModeHTML,
	}
	for _, option := range options {
		option(sendOptions)
	}
	_, err := t.bot.Send(t.chat, msg, sendOptions)
	if err != nil {
		return fmt.Errorf("telegram send message: %w", err)
	}

	return nil
}
