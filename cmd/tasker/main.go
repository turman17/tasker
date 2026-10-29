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
	err = manager.AddTask("Learn python", "New Random info")
	check(err)
	tasks := manager.GetAll()
	fmt.Printf("\n\n")
	for i := range tasks {
		fmt.Printf("Task: %+v\n", tasks[i])
	}
	err = manager.DeleteTask(1)
	check(err)
	err = manager.UpdateTask(0, 1)
	check(err)
	tasks = manager.GetAll()
	fmt.Printf("\n\n")
	for i := range tasks {
		fmt.Printf("Task: %+v\n", tasks[i])
	}
}
