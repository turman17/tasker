package task

import (
	"testing"
	"time"
)

func TestCreateTask(t *testing.T) {
	before := time.Now()

	got, err := createTask("Write tests", "Cover the task package", 7)
	if err != nil {
		t.Fatalf("createTask() unexpected error: %v", err)
	}

	after := time.Now()
	if got.ID != 7 {
		t.Errorf("createTask() ID = %d, want 7", got.ID)
	}
	if got.Title != "Write tests" {
		t.Errorf("createTask() Title = %q, want %q", got.Title, "Write tests")
	}
	if got.Description != "Cover the task package" {
		t.Errorf("createTask() Description = %q, want %q", got.Description, "Cover the task package")
	}
	if got.Status != Todo {
		t.Errorf("createTask() Status = %v, want %v", got.Status, Todo)
	}
	if got.CreatedAt.Before(before) || got.CreatedAt.After(after) {
		t.Errorf("createTask() CreatedAt = %v, want between %v and %v", got.CreatedAt, before, after)
	}
	if !got.DoneAt.IsZero() {
		t.Errorf("createTask() DoneAt = %v, want zero value", got.DoneAt)
	}
}

func TestCreateTaskEmptyTitle(t *testing.T) {
	got, err := createTask("", "description", 0)
	if err == nil {
		t.Fatal("createTask() error = nil, want an error")
	}
	if err.Error() != "title cant be empty" {
		t.Errorf("createTask() error = %q, want %q", err, "title cant be empty")
	}
	if got != (Task{}) {
		t.Errorf("createTask() task = %+v, want zero value", got)
	}
}
