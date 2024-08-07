package db

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/paragor/todo/pkg/models"
	"io"
	"net/http"
)

type remoteRepository struct {
	addr       string
	token      string
	httpClient *http.Client
}

func NewRemoteRepository(addr string, token string, httpClient *http.Client) *remoteRepository {
	return &remoteRepository{addr: addr, httpClient: httpClient, token: token}
}

func (r *remoteRepository) addAuth(request *http.Request) {
	request.Header.Set("Authorization", r.token)
}

func (r *remoteRepository) Ping() error {
	request, err := http.NewRequest("GET", r.addr+"/api/ping", nil)
	if err != nil {
		return fmt.Errorf("cant create request: %w", err)
	}
	r.addAuth(request)
	response, err := r.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("cant connect to remote server: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return fmt.Errorf("unexpected status code from remote server: %d", response.StatusCode)
	}

	return nil
}

func (r *remoteRepository) Get(UUID uuid.UUID) (*models.Task, error) {
	request, err := http.NewRequest("GET", r.addr+"/api/get_task?uuid="+UUID.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("cant create request: %w", err)
	}
	r.addAuth(request)
	response, err := r.httpClient.Do(request)
	defer response.Body.Close()
	if response.StatusCode == 204 {
		return nil, nil
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("cant read data from remote server: status code: %d", response.StatusCode)
	}

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code from remote server: status code %d; pody part: %s", response.StatusCode, string(data[:min(255, len(data))]))
	}

	task := &models.Task{}
	if err := json.Unmarshal(data, task); err != nil {
		return nil, fmt.Errorf("unmarshal error:%w, status code %d; pody part: %s", err, response.StatusCode, string(data[:min(255, len(data))]))
	}

	return task, nil
}

func (r *remoteRepository) Insert(t *models.Task) error {
	requestData, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("cant marshal task: %w", err)
	}
	request, err := http.NewRequest("PUT", r.addr+"/api/insert_task", bytes.NewReader(requestData))
	if err != nil {
		return fmt.Errorf("cant create request: %w", err)
	}
	r.addAuth(request)

	response, err := r.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("cant connect to remote server: %w", err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		if response.StatusCode == 200 {
			return nil
		}
		return fmt.Errorf("cant read data from remote server: status code: %d", response.StatusCode)
	}

	if response.StatusCode != 200 {
		return fmt.Errorf("unexpected status code from remote server: status code %d; pody part: %s", response.StatusCode, string(data[:min(255, len(data))]))
	}

	return nil
}

func (r *remoteRepository) All() ([]*models.Task, error) {
	request, err := http.NewRequest("GET", r.addr+"/api/all", nil)
	if err != nil {
		return nil, fmt.Errorf("cant create request: %w", err)
	}
	r.addAuth(request)
	response, err := r.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("cant connect to remote server: %w", err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("cant read data from remote server: status code: %d", response.StatusCode)
	}

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code from remote server: status code %d; pody part: %s", response.StatusCode, string(data[:min(255, len(data))]))
	}

	tasks := []*models.Task{}
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("unmarshal error:%w, status code %d; pody part: %s", err, response.StatusCode, string(data[:min(255, len(data))]))
	}

	return tasks, nil
}
