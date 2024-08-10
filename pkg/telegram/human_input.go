package telegram

import (
	"fmt"
	"github.com/paragor/todo/pkg/models"
)

func (t *TelegramServer) humanInput(input string) error {
	parsedInput, err := models.ParseHumanInput(input)
	if err != nil {
		return fmt.Errorf("cant parse command: %w", err)
	}

	switch parsedInput.Action {
	case models.HumanActionInfo:
		task, err := t.db.Get(*parsedInput.ActionUUID)
		if err != nil {
			return fmt.Errorf("cant fetch task: %w", err)
		}
		msg, err := renderTemplate("message/task", task)
		if err != nil {
			return fmt.Errorf("cant render template: %w", err)
		}
		err = t.sendMessageHtml(msg, t.withTaskWebApp(task.UUID))
		if err != nil {
			return fmt.Errorf("cant send response (%s): %w", task.UUID, err)
		}
		return nil
	case models.HumanActionAdd:
		task := models.NewTask()
		parsedInput.Options.ModifyTask(task)
		if err := t.db.Insert(task); err != nil {
			return fmt.Errorf("cant insert task: %w", err)
		}
		msg, err := renderTemplate("message/task", task)
		if err != nil {
			return fmt.Errorf("cant render template: %w", err)
		}
		err = t.sendMessageHtml(msg, t.withTaskWebApp(task.UUID))
		if err != nil {
			return fmt.Errorf("cant send response (%s): %w", task.UUID, err)
		}
		return nil
	case models.HumanActionModify, models.HumanActionDone:
		task, err := t.db.Get(*parsedInput.ActionUUID)
		if err != nil {
			return fmt.Errorf("cant fetch task: %w", err)
		}
		parsedInput.Options.ModifyTask(task)
		if err := t.db.Insert(task); err != nil {
			return fmt.Errorf("cant insert task: %w", err)
		}
		msg, err := renderTemplate("message/task", task)
		if err != nil {
			return fmt.Errorf("cant render template: %w", err)
		}
		err = t.sendMessageHtml(msg, t.withTaskWebApp(task.UUID))
		if err != nil {
			return fmt.Errorf("cant send response (%s): %w", task.UUID, err)
		}
		return nil
	case models.HumanActionCopy:
		task, err := t.db.Get(*parsedInput.ActionUUID)
		if err != nil {
			return fmt.Errorf("cant fetch task: %w", err)
		}
		parsedInput.Options.ModifyTask(task)
		task = task.Clone(true)
		if task.Status != models.Pending && parsedInput.Options.Status == nil {
			task.Status = models.Pending
		}
		if err := t.db.Insert(task); err != nil {
			return fmt.Errorf("cant insert task: %w", err)
		}
		msg, err := renderTemplate("message/task", task)
		if err != nil {
			return fmt.Errorf("cant render template: %w", err)
		}
		err = t.sendMessageHtml(msg, t.withTaskWebApp(task.UUID))
		if err != nil {
			return fmt.Errorf("cant send response (%s): %w", task.UUID, err)
		}
		return nil
	case models.HumanActionAgenda:
		if err := t.TriggerAgenda(); err != nil {
			return fmt.Errorf("cant send agenda: %w", err)
		}
		return nil
	case models.HumanActionList:
		tasks, err := t.db.All()
		if err != nil {
			return fmt.Errorf("cant get tasks: %w", err)
		}
		tasks = parsedInput.Options.ToListFilter().Apply(tasks)
		if len(tasks) == 0 {
			err = t.sendMessageHtml("Nothing...", t.withTaskFilterWebApp(parsedInput.Options.ToListFilter()))
			if err != nil {
				return fmt.Errorf("cant send response list: %w", err)
			}
		}
		msg, err := renderTemplate("message/tasks_shortlist", tasks)
		if err != nil {
			return fmt.Errorf("cant render template: %w", err)
		}
		err = t.sendMessageHtml(msg, t.withTaskFilterWebApp(parsedInput.Options.ToListFilter()))
		if err != nil {
			return fmt.Errorf("cant send response list: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unkown action: %s", parsedInput.Action)
	}
}
