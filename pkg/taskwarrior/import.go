package taskwarrior

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/paragor/todo/pkg/models"
	"os"
	"os/exec"
	"slices"
	"time"
)

type ImportConfig struct {
	Filepath      string
	SkipDeleted   bool
	SkipCompleted bool
	SkipRecur     bool
}

func Import(cfg *ImportConfig) ([]*models.Task, error) {
	tasks := []*twTask{}
	var err error
	if len(cfg.Filepath) > 0 {
		data, err := os.ReadFile(cfg.Filepath)
		if err != nil {
			return nil, fmt.Errorf("cant open file %s: %w", cfg.Filepath, err)
		}
		if err := json.Unmarshal(data, &tasks); err != nil {
			return nil, fmt.Errorf("cant unmarshal: %w", err)
		}
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		tasks, err = runTaskExport(ctx)
		if err != nil {
			return nil, fmt.Errorf("cant open file %s: %w", cfg.Filepath, err)
		}
	}
	tasks = slices.DeleteFunc(tasks, func(t *twTask) bool {
		if cfg.SkipDeleted && t.Status == "deleted" {
			return true
		}
		if cfg.SkipCompleted && t.Status == "completed" {
			return true
		}
		if cfg.SkipRecur && len(t.Recur) > 0 {
			return true
		}
		return false
	})
	result := []*models.Task{}
	for _, task := range tasks {
		appTask, err := task.toApplicationTask()
		if err != nil {
			return nil, fmt.Errorf("cant convert task %s: %w", task.Uuid, err)
		}
		result = append(result, appTask)
	}
	return result, nil
}
func runTaskExport(ctx context.Context) ([]*twTask, error) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	cmd := exec.CommandContext(ctx, "task", "export")
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("fail to start: %w. stdout:\n %s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	err = cmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("finish with fail: %w. stdout:\n %s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}

	tasks := []*twTask{}
	if err := json.Unmarshal(stdout.Bytes(), &tasks); err != nil {
		return nil, fmt.Errorf("fail unmarshal: %w. stdout:\n %s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	return tasks, nil
}
