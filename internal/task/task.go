package task

import (
	"strings"
	"time"
)

type Status int

const (
	Todo Status = iota
	InProgress
	Done
)

type Task struct {
	ID          int
	Title       string
	Description string
	CreatedAt   time.Time
	DoneAt      time.Time
	Status      Status
}

func createTask(title, description string, counter int) (Task, error) {
	if strings.TrimSpace(title) == "" {
		return Task{}, ErrInvalidTitle
	}
	task := Task{ID: counter, Title: title,
		Description: description,
		CreatedAt:   time.Now(), Status: Todo}
	return task, nil
}
