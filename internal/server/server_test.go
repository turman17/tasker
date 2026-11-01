package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

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
	server, store := newTestServer()
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
	created, err := store.GetTask(context.Background(), 0)
	if err != nil {
		t.Fatalf("created task was not stored: %v", err)
	}
	if created.Title != "Write tests" || created.Description != "Test the server" {
		t.Errorf("created task = %+v, want submitted title and description", created)
	}
}

func TestCreateTaskErrors(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantBody string
	}{
		{name: "malformed JSON", body: `{"title":`, wantBody: "Invalid JSON request body\n"},
		{name: "multiple JSON values", body: `{"title":"one"} {"title":"two"}`, wantBody: "Request body must contain exactly one JSON value\n"},
		{name: "empty title", body: `{"title":"","description":"missing title"}`, wantBody: "title cannot be empty\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, store := newTestServer()
			request := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(tt.body))
			response := httptest.NewRecorder()

			server.createTask(response, request)

			if response.Code != http.StatusBadRequest {
				t.Errorf("createTask() status = %d, want %d", response.Code, http.StatusBadRequest)
			}
			if got := response.Body.String(); got != tt.wantBody {
				t.Errorf("createTask() body = %q, want %q", got, tt.wantBody)
			}
			tasks, err := store.ListTasks(context.Background())
			if err != nil {
				t.Fatalf("ListTasks() unexpected error: %v", err)
			}
			if len(tasks) != 0 {
				t.Errorf("createTask() stored %d tasks after an error, want 0", len(tasks))
			}
		})
	}
}

func TestGetTasks(t *testing.T) {
	server, store := newTestServer()
	mustCreateTask(t, store, "First", "first task")
	mustCreateTask(t, store, "Second", "second task")
	request := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	response := httptest.NewRecorder()

	server.getTasks(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("getTasks() status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("getTasks() Content-Type = %q, want %q", got, "application/json")
	}
	var tasks []task.Task
	if err := json.NewDecoder(response.Body).Decode(&tasks); err != nil {
		t.Fatalf("decode getTasks() response: %v", err)
	}
	if len(tasks) != 2 || tasks[0].Title != "First" || tasks[1].Title != "Second" {
		t.Errorf("getTasks() returned unexpected tasks: %+v", tasks)
	}
}

func TestGetTask(t *testing.T) {
	server, store := newTestServer()
	mustCreateTask(t, store, "Existing", "task")
	request := requestWithID(http.MethodGet, "/tasks/0", nil, "0")
	response := httptest.NewRecorder()

	server.getTask(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("getTask() status = %d, want %d; body = %q", response.Code, http.StatusOK, response.Body.String())
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
			server, _ := newTestServer()
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
	server, store := newTestServer()
	mustCreateTask(t, store, "Delete me", "task")
	request := requestWithID(http.MethodDelete, "/tasks/0", nil, "0")
	response := httptest.NewRecorder()

	server.deleteTask(response, request)

	if response.Code != http.StatusNoContent {
		t.Errorf("deleteTask() status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if _, err := store.GetTask(context.Background(), 0); err == nil {
		t.Error("deleteTask() left the deleted task in the store")
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
			server, _ := newTestServer()
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
	server, store := newTestServer()
	mustCreateTask(t, store, "Update me", "task")
	request := requestWithID(http.MethodPatch, "/tasks/0", strings.NewReader(`{"status":1}`), "0")
	response := httptest.NewRecorder()

	server.updateStatus(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("updateStatus() status = %d, want %d; body = %q", response.Code, http.StatusNoContent, response.Body.String())
	}
	updated, err := store.GetTask(context.Background(), 0)
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
		{name: "malformed JSON", id: "0", body: `{"status":`, wantStatus: http.StatusBadRequest, wantBody: "Invalid JSON request body\n"},
		{name: "missing status", id: "0", body: `{}`, wantStatus: http.StatusBadRequest, wantBody: "Status is required\n"},
		{name: "invalid ID", id: "abc", body: `{"status":1}`, wantStatus: http.StatusBadRequest, wantBody: "Invalid task ID\n"},
		{name: "task not found", id: "42", body: `{"status":1}`, wantStatus: http.StatusNotFound, wantBody: "task not found\n"},
		{name: "invalid status", id: "0", body: `{"status":7}`, addTask: true, wantStatus: http.StatusBadRequest, wantBody: "invalid task status\n"},
		{name: "same status", id: "0", body: `{"status":0}`, addTask: true, wantStatus: http.StatusBadRequest, wantBody: "invalid task status\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, store := newTestServer()
			if tt.addTask {
				mustCreateTask(t, store, "Existing", "task")
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
	store := newMemoryTaskStore()
	server := NewServer(store)
	if server == nil {
		t.Fatal("NewServer() returned nil")
	}
	if server.store != store {
		t.Fatal("NewServer() did not retain the injected task store")
	}
}

func requestWithID(method, target string, body io.Reader, id string) *http.Request {
	request := httptest.NewRequest(method, target, body)
	request.SetPathValue("id", id)
	return request
}

func newTestServer() (*Server, *memoryTaskStore) {
	store := newMemoryTaskStore()
	return NewServer(store), store
}

func mustCreateTask(t *testing.T, store TaskStore, title, description string) task.Task {
	t.Helper()
	created, err := store.CreateTask(context.Background(), title, description)
	if err != nil {
		t.Fatalf("CreateTask() unexpected error: %v", err)
	}
	return created
}

type memoryTaskStore struct {
	tasks  map[int]task.Task
	nextID int
}

func newMemoryTaskStore() *memoryTaskStore {
	return &memoryTaskStore{tasks: make(map[int]task.Task)}
}

func (s *memoryTaskStore) CreateTask(_ context.Context, title, description string) (task.Task, error) {
	if strings.TrimSpace(title) == "" {
		return task.Task{}, task.ErrInvalidTitle
	}
	created := task.Task{
		ID:          s.nextID,
		Title:       title,
		Description: description,
		CreatedAt:   time.Now(),
		Status:      task.Todo,
	}
	s.tasks[created.ID] = created
	s.nextID++
	return created, nil
}

func (s *memoryTaskStore) GetTask(_ context.Context, id int) (task.Task, error) {
	found, ok := s.tasks[id]
	if !ok {
		return task.Task{}, task.ErrTaskNotFound
	}
	return found, nil
}

func (s *memoryTaskStore) ListTasks(_ context.Context) ([]task.Task, error) {
	ids := make([]int, 0, len(s.tasks))
	for id := range s.tasks {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	tasks := make([]task.Task, 0, len(ids))
	for _, id := range ids {
		tasks = append(tasks, s.tasks[id])
	}
	return tasks, nil
}

func (s *memoryTaskStore) UpdateTaskStatus(_ context.Context, id int, status task.Status) (task.Task, error) {
	found, ok := s.tasks[id]
	if !ok {
		return task.Task{}, task.ErrTaskNotFound
	}
	if status < task.Todo || status > task.Done || status == found.Status {
		return task.Task{}, task.ErrInvalidStatus
	}
	found.Status = status
	if status == task.Done {
		found.DoneAt = time.Now()
	} else {
		found.DoneAt = time.Time{}
	}
	s.tasks[id] = found
	return found, nil
}

func (s *memoryTaskStore) DeleteTask(_ context.Context, id int) (bool, error) {
	if _, ok := s.tasks[id]; !ok {
		return false, nil
	}
	delete(s.tasks, id)
	return true, nil
}
