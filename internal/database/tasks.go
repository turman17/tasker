package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/turman17/tasker/internal/task"
)

const taskColumns = "id, title, description, created_at, done_at, status"

type TaskStore struct {
	db *sql.DB
}

func NewTaskStore(db *sql.DB) *TaskStore {
	return &TaskStore{db: db}
}

func (s *TaskStore) CreateTask(ctx context.Context, title, description string) (task.Task, error) {
	if strings.TrimSpace(title) == "" {
		return task.Task{}, task.ErrInvalidTitle
	}

	query := `
		INSERT INTO tasks (title, description)
		VALUES ($1, $2)
		RETURNING ` + taskColumns

	created, err := scanTask(s.db.QueryRowContext(ctx, query, title, description))
	if err != nil {
		return task.Task{}, fmt.Errorf("create task: %w", err)
	}
	return created, nil
}

func (s *TaskStore) GetTask(ctx context.Context, id int) (task.Task, error) {
	query := `SELECT ` + taskColumns + ` FROM tasks WHERE id = $1`

	found, err := scanTask(s.db.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return task.Task{}, task.ErrTaskNotFound
	}
	if err != nil {
		return task.Task{}, fmt.Errorf("get task %d: %w", id, err)
	}
	return found, nil
}

func (s *TaskStore) ListTasks(ctx context.Context) ([]task.Task, error) {
	query := `SELECT ` + taskColumns + ` FROM tasks ORDER BY id`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]task.Task, 0)
	for rows.Next() {
		current, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scan listed task: %w", err)
		}
		tasks = append(tasks, current)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}
	return tasks, nil
}

func (s *TaskStore) UpdateTaskStatus(ctx context.Context, id int, status task.Status) (task.Task, error) {
	if status < task.Todo || status > task.Done {
		return task.Task{}, task.ErrInvalidStatus
	}

	query := `
		UPDATE tasks
		SET status = $2,
			done_at = CASE WHEN $2 = $3 THEN NOW() ELSE NULL END
		WHERE id = $1 AND status <> $2
		RETURNING ` + taskColumns

	updated, err := scanTask(s.db.QueryRowContext(ctx, query, id, status, task.Done))
	if err == nil {
		return updated, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return task.Task{}, fmt.Errorf("update task %d status: %w", id, err)
	}

	var currentStatus task.Status
	err = s.db.QueryRowContext(ctx, `SELECT status FROM tasks WHERE id = $1`, id).Scan(&currentStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return task.Task{}, task.ErrTaskNotFound
	}
	if err != nil {
		return task.Task{}, fmt.Errorf("check task %d after update: %w", id, err)
	}
	return task.Task{}, task.ErrInvalidStatus
}

func (s *TaskStore) DeleteTask(ctx context.Context, id int) (bool, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return false, fmt.Errorf("delete task %d: %w", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("get deleted task count: %w", err)
	}
	return rowsAffected > 0, nil
}

type taskScanner interface {
	Scan(dest ...any) error
}

func scanTask(scanner taskScanner) (task.Task, error) {
	var result task.Task
	var doneAt sql.NullTime

	err := scanner.Scan(
		&result.ID,
		&result.Title,
		&result.Description,
		&result.CreatedAt,
		&doneAt,
		&result.Status,
	)
	if err != nil {
		return task.Task{}, err
	}
	if doneAt.Valid {
		result.DoneAt = doneAt.Time
	}
	return result, nil
}
