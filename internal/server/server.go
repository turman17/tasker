package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/turman17/tasker/internal/task"
)

type Server struct {
	manager *task.Manager
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
	response := s.manager.GetAll()
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
	task, err := s.manager.GetTask(id)
	if err != nil {
		log.Printf("Failed to fetch task: %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(task); err != nil {
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
	err = s.manager.DeleteTask(id)
	if err != nil {
		log.Printf("Failed to delete task: %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
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
	if err := s.manager.UpdateTask(id, *request.Status); err != nil {
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
	if err := s.manager.AddTask(request.Title, request.Description); err != nil {
		log.Printf("Failed to create task: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Print("New task created")
	w.WriteHeader(http.StatusCreated)
}

func NewServer() *Server {
	manager := task.CreateManager()
	server := Server{
		manager: &manager,
	}
	return &server
}

func (s *Server) StartServer(port int) error {
	mux := http.NewServeMux()
	fmt.Printf("Server started: %d\n", port)

	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("GET /tasks", s.getTasks)
	mux.HandleFunc("GET /tasks/{id}", s.getTask)

	mux.HandleFunc("POST /tasks", s.createTask)

	mux.HandleFunc("DELETE /tasks/{id}", s.deleteTask)
	mux.HandleFunc("PATCH /tasks/{id}", s.updateStatus)

	return http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
}
