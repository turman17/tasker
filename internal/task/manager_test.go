package task

import "testing"

func TestManagerAddAndGetTask(t *testing.T) {
	manager := CreateManager()

	if err := manager.AddTask("First", "first description"); err != nil {
		t.Fatalf("AddTask() unexpected error: %v", err)
	}
	if err := manager.AddTask("Second", "second description"); err != nil {
		t.Fatalf("AddTask() unexpected error: %v", err)
	}

	tests := []struct {
		id          int
		title       string
		description string
	}{
		{id: 0, title: "First", description: "first description"},
		{id: 1, title: "Second", description: "second description"},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			got, err := manager.GetTask(tt.id)
			if err != nil {
				t.Fatalf("GetTask(%d) unexpected error: %v", tt.id, err)
			}
			if got.ID != tt.id {
				t.Errorf("GetTask(%d) ID = %d, want %d", tt.id, got.ID, tt.id)
			}
			if got.Title != tt.title {
				t.Errorf("GetTask(%d) Title = %q, want %q", tt.id, got.Title, tt.title)
			}
			if got.Description != tt.description {
				t.Errorf("GetTask(%d) Description = %q, want %q", tt.id, got.Description, tt.description)
			}
			if got.Status != Todo {
				t.Errorf("GetTask(%d) Status = %v, want %v", tt.id, got.Status, Todo)
			}
		})
	}
}

func TestManagerAddTaskRejectsEmptyTitleWithoutConsumingID(t *testing.T) {
	manager := CreateManager()

	if err := manager.AddTask("", "invalid"); err == nil {
		t.Fatal("AddTask() error = nil, want an error")
	}
	if err := manager.AddTask("Valid", "description"); err != nil {
		t.Fatalf("AddTask() unexpected error: %v", err)
	}

	got, err := manager.GetTask(0)
	if err != nil {
		t.Fatalf("GetTask(0) unexpected error: %v", err)
	}
	if got.Title != "Valid" {
		t.Errorf("GetTask(0) Title = %q, want %q", got.Title, "Valid")
	}
}

func TestManagerGetTaskNotFound(t *testing.T) {
	manager := CreateManager()

	got, err := manager.GetTask(99)
	if err == nil {
		t.Fatal("GetTask() error = nil, want an error")
	}
	if err.Error() != "task not found" {
		t.Errorf("GetTask() error = %q, want %q", err, "task not found")
	}
	if got != (Task{}) {
		t.Errorf("GetTask() task = %+v, want zero value", got)
	}
}

func TestManagerGetAllReturnsCopy(t *testing.T) {
	manager := CreateManager()
	if err := manager.AddTask("Original", "description"); err != nil {
		t.Fatalf("AddTask() unexpected error: %v", err)
	}

	got := manager.GetAll()
	if len(got) != 1 {
		t.Fatalf("GetAll() returned %d tasks, want 1", len(got))
	}

	delete(got, 0)
	got[10] = Task{ID: 10, Title: "Injected"}

	if _, err := manager.GetTask(0); err != nil {
		t.Errorf("modifying GetAll() result removed manager task: %v", err)
	}
	if _, err := manager.GetTask(10); err == nil {
		t.Error("modifying GetAll() result added a task to manager")
	}
}

func TestManagerDeleteTask(t *testing.T) {
	manager := CreateManager()
	if err := manager.AddTask("Delete me", "description"); err != nil {
		t.Fatalf("AddTask() unexpected error: %v", err)
	}

	if err := manager.DeleteTask(0); err != nil {
		t.Fatalf("DeleteTask(0) unexpected error: %v", err)
	}
	if _, err := manager.GetTask(0); err == nil {
		t.Error("GetTask(0) error = nil after deletion, want an error")
	}
	if err := manager.DeleteTask(0); err == nil {
		t.Error("DeleteTask(0) error = nil for missing task, want an error")
	} else if err.Error() != "task not found" {
		t.Errorf("DeleteTask(0) error = %q, want %q", err, "task not found")
	}
}

func TestManagerUpdateTask(t *testing.T) {
	tests := []struct {
		name   string
		status Status
	}{
		{name: "todo", status: Todo},
		{name: "in progress", status: InProgress},
		{name: "done", status: Done},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := CreateManager()
			if err := manager.AddTask("Update me", "description"); err != nil {
				t.Fatalf("AddTask() unexpected error: %v", err)
			}

			if err := manager.UpdateTask(0, int(tt.status)); err != nil {
				t.Fatalf("UpdateTask() unexpected error: %v", err)
			}
			got, err := manager.GetTask(0)
			if err != nil {
				t.Fatalf("GetTask() unexpected error: %v", err)
			}
			if got.Status != tt.status {
				t.Errorf("UpdateTask() Status = %v, want %v", got.Status, tt.status)
			}
		})
	}
}

func TestManagerUpdateTaskErrors(t *testing.T) {
	tests := []struct {
		name      string
		id        int
		status    int
		wantError string
	}{
		{name: "task not found", id: 99, status: int(Todo), wantError: "task not found"},
		{name: "status below range", id: 0, status: int(Todo) - 1, wantError: "invalid task status"},
		{name: "status above range", id: 0, status: int(Done) + 1, wantError: "invalid task status"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := CreateManager()
			if err := manager.AddTask("Original", "description"); err != nil {
				t.Fatalf("AddTask() unexpected error: %v", err)
			}

			err := manager.UpdateTask(tt.id, tt.status)
			if err == nil {
				t.Fatal("UpdateTask() error = nil, want an error")
			}
			if err.Error() != tt.wantError {
				t.Errorf("UpdateTask() error = %q, want %q", err, tt.wantError)
			}

			got, getErr := manager.GetTask(0)
			if getErr != nil {
				t.Fatalf("GetTask(0) unexpected error: %v", getErr)
			}
			if got.Status != Todo {
				t.Errorf("task status changed after failed update: got %v, want %v", got.Status, Todo)
			}
		})
	}
}
