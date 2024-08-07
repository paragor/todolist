package cron

import (
	"context"
	"fmt"
	"sync"
	"time"
)

var (
	CronError = fmt.Errorf("cron error")

	AlreadyRunningError      = fmt.Errorf("%w: cron is already running", CronError)
	TriggerDateIsExpireError = fmt.Errorf("%w: trigger date is expired", CronError)
	ForceStoppedError        = fmt.Errorf("%w: trigger is force stopped", CronError)
)

type Cron struct {
	triggerDate time.Time
	fn          func() error

	m         sync.Mutex
	isStarted bool
	isDone    bool

	ctx    context.Context
	cancel func()
}

func NewCron(triggerDate time.Time, fn func() error) *Cron {
	return &Cron{triggerDate: triggerDate, fn: fn}
}

func (c *Cron) GoRun() (chan error, error) {
	if c.IsStarted() {
		return nil, AlreadyRunningError
	}
	c.m.Lock()
	c.isStarted = true
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.m.Unlock()
	now := time.Now()
	if now.After(c.triggerDate) {
		return nil, TriggerDateIsExpireError
	}
	triggerChan := time.After(c.triggerDate.Sub(now))
	errChan := make(chan error, 1)
	go func() {
		select {
		case <-triggerChan:
			errChan <- c.fn()
		case <-c.ctx.Done():
			errChan <- ForceStoppedError
		}
		c.m.Lock()
		c.isDone = true
		c.m.Unlock()
		close(errChan)
	}()
	return errChan, nil
}

func (c *Cron) Stop() {
	c.m.Lock()
	defer c.m.Unlock()
	if c.cancel != nil {
		c.cancel()
	}
}

func (c *Cron) IsStarted() bool {
	c.m.Lock()
	defer c.m.Unlock()
	return c.isStarted
}
func (c *Cron) IsDone() bool {
	c.m.Lock()
	defer c.m.Unlock()
	return c.isDone
}

func (c *Cron) When() time.Time {
	return c.triggerDate
}
