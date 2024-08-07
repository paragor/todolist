package httpserver

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/paragor/todo/pkg/models"
	"io"
	"net/http"
)

func (h *httpServer) apiPing(writer http.ResponseWriter, request *http.Request) {
	writer.WriteHeader(200)
}

func (h *httpServer) apiInsertTask(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()
	data, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, "cant read request body: "+err.Error(), 400)
		return
	}
	task := &models.Task{}
	if err := json.Unmarshal(data, task); err != nil {
		http.Error(writer, "cant unmarshal task: "+err.Error(), 400)
		return
	}
	if err := task.Validate(); err != nil {
		http.Error(writer, "invalid task: "+err.Error(), 400)
		return
	}
	if err := h.repository.Insert(task); err != nil {
		http.Error(writer, "cant insert task: "+err.Error(), 500)
		return
	}
	writer.WriteHeader(200)
}

func (h *httpServer) apiAllTask(writer http.ResponseWriter, request *http.Request) {
	tasks, err := h.repository.All()
	if err != nil {
		http.Error(writer, "cant get tasks: "+err.Error(), 500)
		return
	}
	response, err := json.Marshal(tasks)
	if err != nil {
		http.Error(writer, "cant marshal task: "+err.Error(), 500)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(200)
	_, _ = writer.Write(response)
}
func (h *httpServer) apiGetTask(writer http.ResponseWriter, request *http.Request) {
	_ = request.ParseForm()
	UUID := request.Form.Get("uuid")
	if len(UUID) == 0 {
		http.Error(writer, "uuid cant not be empty", 400)
		return
	}
	parsedUUID, err := uuid.Parse(UUID)
	if err != nil {
		http.Error(writer, "cant parse UUID: %s"+err.Error(), 400)
		return
	}
	task, err := h.repository.Get(parsedUUID)
	if err != nil {
		http.Error(writer, "cant get task: "+err.Error(), 500)
		return
	}
	if task == nil {
		writer.WriteHeader(204)
		return
	}
	response, err := json.Marshal(task)
	if err != nil {
		http.Error(writer, "cant marshal task: "+err.Error(), 500)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(200)
	_, _ = writer.Write(response)
}
