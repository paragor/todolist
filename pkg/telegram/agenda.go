package telegram

import (
	"fmt"
	"github.com/paragor/todo/pkg/models"
)

func (t *TelegramServer) TriggerAgenda() error {
	if t.bot == nil {
		return fmt.Errorf("server is not started")
	}
	tasks, err := t.db.All()
	if err != nil {
		return fmt.Errorf("cant get tasks list: %w", err)
	}
	tasks = models.NewDefaultListFilter().Apply(tasks)
	agenda := models.Agenda(tasks)
	msg, err := renderTemplate("message/agenda", agenda)
	if err != nil {
		return fmt.Errorf("cant render template: %w", err)
	}
	if err := t.sendMessageHtml(msg, t.withAgendaWebApp(), t.withEnableNotifications()); err != nil {
		return fmt.Errorf("cant send telegram msg: %w", err)
	}
	return nil
}
