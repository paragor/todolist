package events

import (
	"github.com/google/uuid"
	"github.com/paragor/todo/pkg/models"
	"slices"
	"sync"
)

var onDatabaseChangeSubscribers []DatabaseChangeSubscriber

type DatabaseChangeSubscriber interface {
	OnDatabaseChange()
}

var m sync.Mutex

func RegisterOnDatabaseChangeSubscriber(subscriber DatabaseChangeSubscriber) {
	m.Lock()
	defer m.Unlock()
	onDatabaseChangeSubscribers = append(onDatabaseChangeSubscribers, subscriber)
}
func UnRegisterOnDatabaseChangeSubscriber(subscriber DatabaseChangeSubscriber) {
	m.Lock()
	defer m.Unlock()
	onDatabaseChangeSubscribers = slices.DeleteFunc(onDatabaseChangeSubscribers, func(existing DatabaseChangeSubscriber) bool {
		return existing == subscriber
	})
}

type spyRepository struct {
	db models.Repository
}

func NewSpyRepository(db models.Repository) *spyRepository {
	return &spyRepository{db: db}
}

func (s *spyRepository) Get(UUID uuid.UUID) (*models.Task, error) {
	return s.db.Get(UUID)
}

func (s *spyRepository) Insert(t *models.Task) error {
	err := s.db.Insert(t)
	if err == nil {
		for _, subscriber := range onDatabaseChangeSubscribers {
			subscriber.OnDatabaseChange()
		}
	}
	return err
}

func (s *spyRepository) All() ([]*models.Task, error) {
	return s.db.All()
}
