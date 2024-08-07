package telegram

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/paragor/todo/pkg/cron"
	"github.com/paragor/todo/pkg/events"
	"github.com/paragor/todo/pkg/models"
	"log"
	"sync"
	"time"
)

type Notifier struct {
	notifyState    map[uuid.UUID]*cron.Cron
	notifyErrChan  chan error
	refreshErrChan chan error
	db             models.Repository
	telegram       *TelegramServer
	m              sync.Mutex
}

func newNotifier(db models.Repository, telegram *TelegramServer) *Notifier {
	return &Notifier{
		notifyState:    map[uuid.UUID]*cron.Cron{},
		notifyErrChan:  make(chan error, 1000),
		refreshErrChan: make(chan error, 1),
		db:             db,
		telegram:       telegram,
	}
}

func (n *Notifier) Start(ctx context.Context) error {
	err := n.refreshState()
	if err != nil {
		n.close()
		return err
	}
	events.RegisterOnDatabaseChangeSubscriber(n)
	defer events.UnRegisterOnDatabaseChangeSubscriber(n)
	for {
		select {
		case err := <-n.notifyErrChan:
			if err != nil {
				return fmt.Errorf("cron job return error: %w", err)
			}
		case err := <-n.refreshErrChan:
			if err != nil {
				return fmt.Errorf("refresh state return error: %w", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (n *Notifier) OnDatabaseChange() {
	err := n.refreshState()
	if err != nil {
		n.notifyErrChan <- err
	}
}

func (n *Notifier) refreshState() error {
	n.m.Lock()
	defer n.m.Unlock()
	tasks, err := n.db.All()
	if err != nil {
		return fmt.Errorf("cant get task list: %w", err)
	}
	tasks = models.NewDefaultListFilter().Apply(tasks)
	newState := n.createNotifyState(tasks)
	for UUID, oldCron := range n.notifyState {
		newCron, ok := newState[UUID]
		if !ok {
			oldCron.Stop()
		} else {
			if oldCron.When().Equal(newCron.When()) {
				newState[UUID] = oldCron
				if !oldCron.IsStarted() && !oldCron.IsDone() {
					//goland:noinspection GoErrorStringFormat
					return fmt.Errorf("why old cron is not running?")
				}
			} else {
				if err := n.runCron(newCron); err != nil {
					return fmt.Errorf("cant spawn change state cron: %w", err)
				}
				oldCron.Stop()
			}
		}
	}

	for UUID, newCron := range newState {
		if _, ok := n.notifyState[UUID]; !ok {
			if err := n.runCron(newCron); err != nil {
				return fmt.Errorf("cant spawn new state cron: %w", err)
			}
		}
	}

	n.notifyState = newState
	return nil
}

func (n *Notifier) runCron(c *cron.Cron) error {
	errChan, err := c.GoRun()
	if err != nil {
		return err
	}
	go func() {
		err := <-errChan
		if errors.Is(err, cron.ForceStoppedError) {
			return
		}
		n.notifyErrChan <- err
	}()

	return nil
}

func (n *Notifier) close() {
	for _, c := range n.notifyState {
		c.Stop()
	}
}

func (n *Notifier) createNotifyState(tasks []*models.Task) map[uuid.UUID]*cron.Cron {
	result := map[uuid.UUID]*cron.Cron{}

	for _, t := range tasks {
		notifyDate := t.Notify
		if notifyDate == nil || time.Now().After(*notifyDate) {
			continue
		}
		if t.Status != models.Pending {
			continue
		}
		UUID := t.UUID
		result[UUID] = cron.NewCron(*notifyDate, func() error {
			return n.triggerNotify(UUID)
		})
	}
	return result
}

func (n *Notifier) triggerNotify(UUID uuid.UUID) error {
	task, err := n.db.Get(UUID)
	if err != nil {
		return fmt.Errorf("on search task (%s): %w", UUID, err)
	}
	if task == nil {
		log.Printf("try to notify about task %s, but it is not found", UUID)
		return nil
	}
	if task.Status != models.Pending {
		log.Printf("try to notify about task %s, but it is has not pendig status: %s", UUID, task.Status)
		return nil
	}

	log.Printf("notify %s task (%s)", UUID, task.Description)

	msg, err := renderTemplate("message/task", task)
	if err != nil {
		return fmt.Errorf("cant render template: %w", err)
	}
	err = n.telegram.sendMessageHtml(msg, n.telegram.withEnableNotifications(), n.telegram.withTaskWebApp(UUID))
	if err != nil {
		return fmt.Errorf("cant send notify (%s): %w", UUID, err)
	}
	return nil
}
