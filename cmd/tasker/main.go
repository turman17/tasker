package main

import (
	"fmt"
	"os"

	"github.com/turman17/tasker/internal/task"
)

func check(err error) {
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	manager := task.CreateManager()
	err := manager.AddTask("Learn go", "Random info")
	check(err)
	newTask, err := manager.GetTask(0)
	check(err)
	fmt.Printf("%+v\n", newTask)
}
