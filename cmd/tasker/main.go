package main

import (
	"fmt"
	"os"

	"github.com/turman17/tasker/internal/server"
)

func check(err error) {
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}


func main() {
	serv := server.NewServer()
	check(serv.StartServer(5001))
	
}
