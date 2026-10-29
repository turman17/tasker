package task

import (
	"errors"
)

type Manager struct {
	tasks   map[int]Task
	counter int
}

func CreateManager() Manager {
	return Manager{tasks: make(map[int]Task)}
}

func (m *Manager) AddTask(title, description string) error {
	task, err := createTask(title, description, m.counter)
	if err != nil {
		return err
	}
	m.counter++
	m.tasks[task.ID] = task
	return nil
}

func (m *Manager) GetTask(id int) (Task, error) {
	task, exists := m.tasks[id]
	if !exists {
		return Task{}, errors.New("task not found")
	}
	return task, nil
}

func (m *Manager) GetAll() map[int]Task {
	newMap := make(map[int]Task)
	for i := range(m.tasks){
		newMap[i] = m.tasks[i]
	}
	return newMap
}

func (m *Manager) DeleteTask(id int) error {
	_, exists := m.tasks[id]
	if !exists {
		return errors.New("task not found")
	}
	delete(m.tasks, id)
	return nil
}