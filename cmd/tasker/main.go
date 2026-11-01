package main

import (
	"log"
	"os"

	"github.com/turman17/tasker/internal/database"
	"github.com/turman17/tasker/internal/server"
)

func run() error {
	db, err := database.GetConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	log.Print("Database connection established")

	store := database.NewTaskStore(db)
	serv := server.NewServer(store)
	return serv.StartServer(5001)
}

func main() {
	if err := run(); err != nil {
		log.Printf("Application failed: %v", err)
		os.Exit(1)
	}
}
