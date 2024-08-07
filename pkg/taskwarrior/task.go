package taskwarrior

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/paragor/todo/pkg/models"
	"time"
)

type twTask struct {
	Id          int      `json:"id"`
	Description string   `json:"description"`
	Due         string   `json:"due,omitempty"`
	Notify      string   `json:"notify,omitempty"`
	End         string   `json:"end,omitempty"`
	Entry       string   `json:"entry"`
	Modified    string   `json:"modified"`
	Status      string   `json:"status"`
	Uuid        string   `json:"uuid"`
	Urgency     float64  `json:"urgency"`
	Tags        []string `json:"tags,omitempty"`
	Project     string   `json:"project,omitempty"`
	Wait        string   `json:"wait,omitempty"`
	Recur       string   `json:"recur,omitempty"`
}

var (
	uuidError error = fmt.Errorf("invalid uuid")
)

func (t *twTask) toApplicationTask() (*models.Task, error) {
	UUID, err := uuid.Parse(t.Uuid)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", uuidError, err)
	}

	parsedStatus, err := models.NewTaskStatus(t.Status)
	if err != nil {
		parsedStatus = models.Pending
	}

	result := &models.Task{
		UUID:        UUID,
		Description: t.Description,
		Project:     t.Project,
		Tags:        t.Tags,
		Status:      parsedStatus,
		Due:         formatDate(t.Due),
		Notify:      formatDate(t.Notify),
		CreatedAt:   time.Now(),
	}
	parsedCreatedAt := formatDate(t.Entry)
	if parsedCreatedAt != nil {
		result.CreatedAt = *parsedCreatedAt
	}
	return result, nil
}

func formatDate(timeString string) *time.Time {
	parsedTime, _ := time.ParseInLocation("20060102T150405Z", timeString, time.UTC)
	if parsedTime.IsZero() {
		return nil
	}
	parsedTime = parsedTime.In(time.Local)
	return &parsedTime
}
