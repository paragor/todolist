package cron

import (
	"context"
	"fmt"
	"time"
)

type RepeatableCron struct {
	fn             func() error
	nextNotifyTime func() time.Time

	cancel func()
}

func NewRepeatableCron(fn func() error, nextNotifyTime func() time.Time) *RepeatableCron {
	return &RepeatableCron{fn: fn, nextNotifyTime: nextNotifyTime}
}

func (r *RepeatableCron) Start(ctx context.Context, stopper chan<- error) error {
	ctx, r.cancel = context.WithCancel(ctx)
	cron := NewCron(r.nextNotifyTime(), r.fn)
	result, err := cron.GoRun()
	if err != nil {
		return fmt.Errorf("cant start cron: %w", err)
	}
	go func() {
		for {
			select {
			case cronErr := <-result:
				close(result)
				if cronErr != nil {
					stopper <- fmt.Errorf("cron error: %w", cronErr)
					return
				}
				cron = NewCron(r.nextNotifyTime(), r.fn)
				result, err = cron.GoRun()
				if err != nil {
					stopper <- fmt.Errorf("cron start error: %w", cronErr)
					return
				}
			case <-ctx.Done():
				cron.Stop()
				return
			}
		}
	}()
	return nil
}

func (r *RepeatableCron) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
}

func RepeatEveryDayAt(repeatTime time.Time) func() time.Time {
	return func() time.Time {
		now := time.Now()
		notifyAt := time.Date(
			now.Year(),
			now.Month(),
			now.Day(),
			repeatTime.Hour(),
			repeatTime.Minute(),
			repeatTime.Second(),
			repeatTime.Nanosecond(),
			repeatTime.Location(),
		)
		if now.After(notifyAt) {
			notifyAt = notifyAt.Add(24 * time.Hour)
		}
		return notifyAt
	}
}
