package server

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strconv"

	"github.com/turman17/tasker/internal/task"
)

//go:embed static
var staticFiles embed.FS

type TaskStore interface {
	CreateTask(ctx context.Context, title, description string) (task.Task, error)
	GetTask(ctx context.Context, id int) (task.Task, error)
	ListTasks(ctx context.Context) ([]task.Task, error)
	UpdateTaskStatus(ctx context.Context, id int, status task.Status) (task.Task, error)
	DeleteTask(ctx context.Context, id int) (bool, error)
}

type Server struct {
	store TaskStore
}

type createTaskRequest struct {
	Title       string
	Description string
}

type updateTaskStatus struct {
	Status *int
}

func health(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Server is healthy!")
}

func (s *Server) getTasks(w http.ResponseWriter, r *http.Request) {
	response, err := s.store.ListTasks(r.Context())
	if err != nil {
		log.Printf("Failed to list tasks: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode tasks: %v", err)
	}
}

func (s *Server) getTask(w http.ResponseWriter, r *http.Request) {
	rawID := r.PathValue("id")
	id, err := strconv.Atoi(rawID)
	if err != nil {
		log.Printf("Failed to convert an id: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	found, err := s.store.GetTask(r.Context(), id)
	if err != nil {
		log.Printf("Failed to fetch task: %v", err)
		if errors.Is(err, task.ErrTaskNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(found); err != nil {
		log.Printf("Failed to encode task: %v", err)
	}
}

func (s *Server) deleteTask(w http.ResponseWriter, r *http.Request) {
	rawID := r.PathValue("id")
	id, err := strconv.Atoi(rawID)
	if err != nil {
		log.Printf("Failed to convert an id: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	deleted, err := s.store.DeleteTask(r.Context(), id)
	if err != nil {
		log.Printf("Failed to delete task: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !deleted {
		http.Error(w, task.ErrTaskNotFound.Error(), http.StatusNotFound)
		return
	}
	log.Printf("Deleted task: %d", id)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) updateStatus(w http.ResponseWriter, r *http.Request) {
	var request updateTaskStatus
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Printf("Failed to decode task request: %v", err)
		http.Error(w, "Invalid JSON request body", http.StatusBadRequest)
		return
	}
	if request.Status == nil {
		http.Error(w, "Status is required", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}
	if _, err := s.store.UpdateTaskStatus(r.Context(), id, task.Status(*request.Status)); err != nil {
		switch {
		case errors.Is(err, task.ErrTaskNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, task.ErrInvalidStatus):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			log.Printf("Failed to update task: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}
	log.Printf("Updated task %d with status %d", id, *request.Status)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	var request createTaskRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		log.Printf("Failed to decode task request: %v", err)
		http.Error(w, "Invalid JSON request body", http.StatusBadRequest)
		return
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		http.Error(w, "Request body must contain exactly one JSON value", http.StatusBadRequest)
		return
	}
	created, err := s.store.CreateTask(r.Context(), request.Title, request.Description)
	if err != nil {
		log.Printf("Failed to create task: %v", err)
		if errors.Is(err, task.ErrInvalidTitle) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}
	log.Print("New task created")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(created); err != nil {
		log.Printf("Failed to encode task: %v", err)
	}
}

func NewServer(store TaskStore) *Server {
	return &Server{store: store}
}

func (s *Server) StartServer(port int) error {
	mux := http.NewServeMux()
	fmt.Printf("Server started: %d\n", port)

	static, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return err
	}
	mux.Handle("GET /", http.FileServerFS(static))

	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("GET /tasks", s.getTasks)
	mux.HandleFunc("GET /tasks/{id}", s.getTask)

	mux.HandleFunc("POST /tasks", s.createTask)

	mux.HandleFunc("DELETE /tasks/{id}", s.deleteTask)
	mux.HandleFunc("PATCH /tasks/{id}", s.updateStatus)

	return http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
}
