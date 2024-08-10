package telegram

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/paragor/todo/pkg/httpserver"
	"github.com/paragor/todo/pkg/models"
	tele "gopkg.in/telebot.v3"
)

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
func (t *TelegramServer) withTaskFilterWebApp(filter *models.ListFilter) sendOption {
	return func(o *tele.SendOptions) {
		reply := &tele.ReplyMarkup{}
		reply.Inline(
			reply.Split(
				1,
				[]tele.Btn{
					reply.WebApp(
						"List in webapp",
						&tele.WebApp{URL: t.serverPublicUrl + "/?" + httpserver.ListFilterToQuery(filter).Encode()},
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
