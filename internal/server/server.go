package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/turman17/tasker/internal/task"
)

type Server struct {
	manager *task.Manager
}

func health(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Server is healthy!")
}

func (s *Server) getTasks(w http.ResponseWriter, r *http.Request) {
	response := s.manager.GetAll()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Println("Failed to encode tasks:", err)
	}
}

func NewServer() *Server {
	manager := task.CreateManager()
	server := Server{
		manager: &manager,
	}
	return &server
}

func (s *Server) StartServer(port int) error {
	fmt.Printf("Server started: %d\n", port)
	http.HandleFunc("GET /health", health)
	http.HandleFunc("GET /tasks", s.getTasks)

	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
