package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/turman17/tasker/internal/task"
)

func TestHealth(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	health(response, request)

	if response.Code != http.StatusOK {
		t.Errorf("health() status = %d, want %d", response.Code, http.StatusOK)
	}
	if got, want := response.Body.String(), "Server is healthy!"; got != want {
		t.Errorf("health() body = %q, want %q", got, want)
	}
}

func TestCreateTask(t *testing.T) {
	server := NewServer()
	request := httptest.NewRequest(
		http.MethodPost,
		"/tasks",
		strings.NewReader(`{"title":"Write tests","description":"Test the server"}`),
	)
	response := httptest.NewRecorder()

	server.createTask(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("createTask() status = %d, want %d; body = %q", response.Code, http.StatusCreated, response.Body.String())
	}
	created, err := server.manager.GetTask(0)
	if err != nil {
		t.Fatalf("created task was not stored: %v", err)
	}
	if created.Title != "Write tests" {
		t.Errorf("created task title = %q, want %q", created.Title, "Write tests")
	}
	if created.Description != "Test the server" {
		t.Errorf("created task description = %q, want %q", created.Description, "Test the server")
	}
}

func TestCreateTaskErrors(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantBody string
	}{
		{
			name:     "malformed JSON",
			body:     `{"title":`,
			wantBody: "Invalid JSON request body\n",
		},
		{
			name:     "empty title",
			body:     `{"title":"","description":"missing title"}`,
			wantBody: "title cant be empty\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer()
			request := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(tt.body))
			response := httptest.NewRecorder()

			server.createTask(response, request)

			if response.Code != http.StatusBadRequest {
				t.Errorf("createTask() status = %d, want %d", response.Code, http.StatusBadRequest)
			}
			if got := response.Body.String(); got != tt.wantBody {
				t.Errorf("createTask() body = %q, want %q", got, tt.wantBody)
			}
			if got := len(server.manager.GetAll()); got != 0 {
				t.Errorf("createTask() stored %d tasks after an error, want 0", got)
			}
		})
	}
}

func TestGetTasks(t *testing.T) {
	server := NewServer()
	if err := server.manager.AddTask("First", "first task"); err != nil {
		t.Fatalf("AddTask() unexpected error: %v", err)
	}
	if err := server.manager.AddTask("Second", "second task"); err != nil {
		t.Fatalf("AddTask() unexpected error: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	response := httptest.NewRecorder()

	server.getTasks(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("getTasks() status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("getTasks() Content-Type = %q, want %q", got, "application/json")
	}
	var tasks map[int]task.Task
	if err := json.NewDecoder(response.Body).Decode(&tasks); err != nil {
		t.Fatalf("decode getTasks() response: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("getTasks() returned %d tasks, want 2", len(tasks))
	}
	if tasks[0].Title != "First" || tasks[1].Title != "Second" {
		t.Errorf("getTasks() returned unexpected tasks: %+v", tasks)
	}
}

func TestGetTask(t *testing.T) {
	server := NewServer()
	if err := server.manager.AddTask("Existing", "task"); err != nil {
		t.Fatalf("AddTask() unexpected error: %v", err)
	}
	request := requestWithID(http.MethodGet, "/tasks/0", nil, "0")
	response := httptest.NewRecorder()

	server.getTask(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("getTask() status = %d, want %d; body = %q", response.Code, http.StatusOK, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("getTask() Content-Type = %q, want %q", got, "application/json")
	}
	var got task.Task
	if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
		t.Fatalf("decode getTask() response: %v", err)
	}
	if got.ID != 0 || got.Title != "Existing" {
		t.Errorf("getTask() task = %+v, want ID 0 and title %q", got, "Existing")
	}
}

func TestGetTaskErrors(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		wantStatus int
	}{
		{name: "invalid ID", id: "abc", wantStatus: http.StatusBadRequest},
		{name: "not found", id: "42", wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer()
			request := requestWithID(http.MethodGet, "/tasks/"+tt.id, nil, tt.id)
			response := httptest.NewRecorder()

			server.getTask(response, request)

			if response.Code != tt.wantStatus {
				t.Errorf("getTask() status = %d, want %d", response.Code, tt.wantStatus)
			}
		})
	}
}

func TestDeleteTask(t *testing.T) {
	server := NewServer()
	if err := server.manager.AddTask("Delete me", "task"); err != nil {
		t.Fatalf("AddTask() unexpected error: %v", err)
	}
	request := requestWithID(http.MethodDelete, "/tasks/0", nil, "0")
	response := httptest.NewRecorder()

	server.deleteTask(response, request)

	if response.Code != http.StatusNoContent {
		t.Errorf("deleteTask() status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if _, err := server.manager.GetTask(0); err == nil {
		t.Error("deleteTask() left the deleted task in the manager")
	}
}

func TestDeleteTaskErrors(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		wantStatus int
	}{
		{name: "invalid ID", id: "abc", wantStatus: http.StatusBadRequest},
		{name: "not found", id: "42", wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer()
			request := requestWithID(http.MethodDelete, "/tasks/"+tt.id, nil, tt.id)
			response := httptest.NewRecorder()

			server.deleteTask(response, request)

			if response.Code != tt.wantStatus {
				t.Errorf("deleteTask() status = %d, want %d", response.Code, tt.wantStatus)
			}
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	server := NewServer()
	if err := server.manager.AddTask("Update me", "task"); err != nil {
		t.Fatalf("AddTask() unexpected error: %v", err)
	}
	request := requestWithID(http.MethodPatch, "/tasks/0", strings.NewReader(`{"status":1}`), "0")
	response := httptest.NewRecorder()

	server.updateStatus(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("updateStatus() status = %d, want %d; body = %q", response.Code, http.StatusNoContent, response.Body.String())
	}
	updated, err := server.manager.GetTask(0)
	if err != nil {
		t.Fatalf("GetTask() unexpected error: %v", err)
	}
	if updated.Status != task.InProgress {
		t.Errorf("updated status = %v, want %v", updated.Status, task.InProgress)
	}
}

func TestUpdateStatusErrors(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		body       string
		addTask    bool
		wantStatus int
		wantBody   string
	}{
		{
			name:       "malformed JSON",
			id:         "0",
			body:       `{"status":`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "Invalid JSON request body\n",
		},
		{
			name:       "missing status",
			id:         "0",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "Status is required\n",
		},
		{
			name:       "invalid ID",
			id:         "abc",
			body:       `{"status":1}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "Invalid task ID\n",
		},
		{
			name:       "task not found",
			id:         "42",
			body:       `{"status":1}`,
			wantStatus: http.StatusNotFound,
			wantBody:   "task not found\n",
		},
		{
			name:       "invalid status",
			id:         "0",
			body:       `{"status":7}`,
			addTask:    true,
			wantStatus: http.StatusBadRequest,
			wantBody:   "invalid task status\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer()
			if tt.addTask {
				if err := server.manager.AddTask("Existing", "task"); err != nil {
					t.Fatalf("AddTask() unexpected error: %v", err)
				}
			}
			request := requestWithID(http.MethodPatch, "/tasks/"+tt.id, strings.NewReader(tt.body), tt.id)
			response := httptest.NewRecorder()

			server.updateStatus(response, request)

			if response.Code != tt.wantStatus {
				t.Errorf("updateStatus() status = %d, want %d", response.Code, tt.wantStatus)
			}
			if got := response.Body.String(); got != tt.wantBody {
				t.Errorf("updateStatus() body = %q, want %q", got, tt.wantBody)
			}
		})
	}
}

func TestNewServer(t *testing.T) {
	server := NewServer()
	if server == nil {
		t.Fatal("NewServer() returned nil")
	}
	if server.manager == nil {
		t.Fatal("NewServer() manager is nil")
	}
	if got := len(server.manager.GetAll()); got != 0 {
		t.Errorf("NewServer() manager contains %d tasks, want 0", got)
	}
}

func requestWithID(method, target string, body io.Reader, id string) *http.Request {
	request := httptest.NewRequest(method, target, body)
	request.SetPathValue("id", id)
	return request
}
