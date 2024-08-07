package models

import (
	"github.com/google/uuid"
)

type Repository interface {
	Get(UUID uuid.UUID) (*Task, error)
	Insert(t *Task) error
	All() ([]*Task, error)
}
