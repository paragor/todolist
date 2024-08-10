package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/paragor/todo/pkg/models"
	"io"
	"log"
	"os"
	"sync"
)

type DatabaseInternal struct {
	Version int                       `json:"version"`
	Tasks   map[uuid.UUID]models.Task `json:"tasks"`
}

type inMemoryTasksRepository struct {
	filepath          string
	db                *DatabaseInternal
	inProgressWriters sync.WaitGroup
	ctx               context.Context
	cancel            func()

	writeMutex sync.Mutex
}

func NewInMemoryTasksRepository(filepath string) *inMemoryTasksRepository {
	return &inMemoryTasksRepository{filepath: filepath}
}

func (r *inMemoryTasksRepository) Get(UUID uuid.UUID) (*models.Task, error) {
	if task, ok := r.db.Tasks[UUID]; ok {
		return task.Clone(false), nil
	}
	return nil, nil
}

func (r *inMemoryTasksRepository) Insert(task *models.Task) error {
	if r.ctx == nil {
		return fmt.Errorf("repository is not started")
	}
	select {
	case <-r.ctx.Done():
		return fmt.Errorf("repository is closed")
	default:
		r.inProgressWriters.Add(1)
		defer r.inProgressWriters.Done()
	}
	r.db.Version++
	task.Unify()
	if err := task.Validate(); err != nil {
		return fmt.Errorf("invalid task: %w", err)
	}

	r.db.Tasks[task.UUID] = *task.Clone(false)
	return r.flush()
}

func (r *inMemoryTasksRepository) flush() error {
	r.writeMutex.Lock()
	defer r.writeMutex.Unlock()
	data, err := json.Marshal(r.db)
	if err != nil {
		err := fmt.Errorf("cant marhshal database: %w", err)
		log.Println(err)
		r.Stop()
		return err
	}
	tmpFileName := r.filepath + ".new"
	if err := os.WriteFile(tmpFileName, data, 0644); err != nil {
		return fmt.Errorf("cant write to file: %w", err)
	}
	if err := os.Rename(tmpFileName, r.filepath); err != nil {
		return fmt.Errorf("cant rename tmp file to final: %w", err)
	}
	return nil
}

func (r *inMemoryTasksRepository) All() ([]*models.Task, error) {
	result := []*models.Task{}
	for _, t := range r.db.Tasks {
		result = append(result, t.Clone(false))
	}
	models.SortTasks(result)
	return result, nil
}

func (r *inMemoryTasksRepository) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
	r.inProgressWriters.Wait()
}
func (r *inMemoryTasksRepository) Start(ctx context.Context, stopper chan<- error) error {
	f, err := os.Open(r.filepath)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		r.db = &DatabaseInternal{Version: 0, Tasks: map[uuid.UUID]models.Task{}}
	} else if err != nil {
		return fmt.Errorf("cant open file: %w", err)
	} else {
		data, err := io.ReadAll(f)
		if err != nil {
			_ = f.Close()
			return fmt.Errorf("cant read file: %w", err)
		}
		db := &DatabaseInternal{}
		if err := json.Unmarshal(data, &db); err != nil {
			_ = f.Close()
			return fmt.Errorf("cant unmarshal database: %w", err)
		}
		for UUID, task := range db.Tasks {
			if err := task.Validate(); err != nil {
				return fmt.Errorf("onload: invalid task: %s: %w", task.UUID.String(), err)
			}
			if UUID != task.UUID {
				return fmt.Errorf("onload: task uuid and key in struct is not equal: %s", task.UUID.String())
			}
		}
		r.db = db
		_ = f.Close()
	}
	r.ctx, r.cancel = context.WithCancel(ctx)
	go func() {
		select {
		case <-r.ctx.Done():
			err := r.ctx.Err()
			stopper <- fmt.Errorf("stop db.inmemory: %w", err)
		}
	}()
	return nil
}
