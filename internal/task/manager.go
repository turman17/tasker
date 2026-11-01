package task

import (
	"errors"
	"sync"
	"time"
)

type Manager struct {
	tasks   map[int]Task
	counter int
	mutex   sync.Mutex
}

var (
	ErrTaskNotFound = errors.New("task not found")
	ErrInvalidStatus = errors.New("invalid task status")
	ErrInvalidTitle = errors.New("title cannot be empty")
)

func CreateManager() Manager {
	return Manager{tasks: make(map[int]Task)}
}

func (m *Manager) AddTask(title, description string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	task, err := createTask(title, description, m.counter)
	if err != nil {
		return err
	}
	m.counter++
	m.tasks[task.ID] = task
	return nil
}

func (m *Manager) GetTask(id int) (Task, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	task, exists := m.tasks[id]
	if !exists {
		return Task{}, ErrTaskNotFound
	}
	return task, nil
}

func (m *Manager) GetAll() map[int]Task {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	newMap := make(map[int]Task)
	for i := range m.tasks {
		newMap[i] = m.tasks[i]
	}
	return newMap
}

func (m *Manager) DeleteTask(id int) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	_, exists := m.tasks[id]
	if !exists {
		return ErrTaskNotFound
	}
	delete(m.tasks, id)
	return nil
}

func (m *Manager) UpdateTask(id, status int) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	task, exists := m.tasks[id]
	if !exists {
		return ErrTaskNotFound
	}
	if status < int(Todo) || status > int(Done) || status == int(task.Status) {
		return ErrInvalidStatus
	}
	if status == int(Done){
		task.DoneAt = time.Now()
	} else {
		task.DoneAt = time.Time{}
	}
	task.Status = Status(status)
	m.tasks[id] = task

	return nil
}
