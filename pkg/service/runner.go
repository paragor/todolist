package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const defaultShutdownTimeout = 30 * time.Second // Default graceful shutdown timeout in Kubernetes.

type Runnable interface {
	Start(ctx context.Context, stopper chan<- error) error
	Stop()
}

func NewRunner() *Runner {
	r := &Runner{
		shutdownTimeout: defaultShutdownTimeout,
	}

	return r
}

type Runner struct {
	shutdownTimeout time.Duration
	cancel          context.CancelFunc
}

func (r *Runner) Run(services ...Runnable) <-chan error {
	ctx, cancel := signal.NotifyContext(context.Background(), []os.Signal{syscall.SIGTERM, syscall.SIGINT}...)
	r.cancel = cancel

	// leak, but it ok
	stopChan := make(chan error, len(services))
	go func() {
		anyErr, ok := <-stopChan
		if !ok {
			return
		}
		msg := "service.Runner: get any error"
		if anyErr != nil {
			msg += ": " + anyErr.Error()
		}
		log.Println(msg)
		r.Shutdown()
		for err := range stopChan {
			if err != nil {
				log.Printf("service.Runner: stopChan errors: %s", err.Error())
			}
		}
	}()
	resCh := make(chan error, 1)
	for i, service := range services {
		if err := service.Start(ctx, stopChan); err != nil && !errors.Is(err, context.Canceled) {
			r.stopServices(services[:i])
			resCh <- fmt.Errorf("service.Runner: start service: %w", err)
			return resCh
		}

		if ctx.Err() != nil {
			r.stopServices(services[:i])
			close(resCh)
			return resCh
		}
	}

	go func() {
		<-ctx.Done()
		r.stopServices(services)
		close(resCh)
	}()

	return resCh
}

func (r *Runner) Shutdown() {
	if r.cancel != nil {
		r.cancel()
	}
}

func (r *Runner) stopServices(services []Runnable) {
	shutdownTimer := time.NewTimer(r.shutdownTimeout)
	defer shutdownTimer.Stop()

	stoppingCh := make(chan struct{})
	go func() {
		defer close(stoppingCh)
		for i := len(services) - 1; i >= 0; i-- {
			log.Printf("service.Runner: stopping %T", services[i])
			services[i].Stop()
			log.Printf("service.Runner: stopped %T", services[i])
		}
	}()

	select {
	case <-shutdownTimer.C:
		log.Println("service.Runner: shutdown timeout exceeded")
	case <-stoppingCh:
		log.Println("service.Runner: all services are stopped")
	}
}
