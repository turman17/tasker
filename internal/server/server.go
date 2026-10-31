package server

import (
	"encoding/json"
	"fmt"
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

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	var request createTaskRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Invalid JSON request body", http.StatusBadRequest)
		log.Printf("Failed to decode task request: %v", err)
		return
	}
	err = s.manager.AddTask(request.Title, request.Description)
	if err != nil {
		log.Printf("Failed to create task: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("New task created")
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

	return http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
}
