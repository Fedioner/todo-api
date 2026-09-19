package todo

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrNotFound = errors.New("task not found")

type Task struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Ready(ctx context.Context) error {
	var writable, tasksExist bool
	err := s.db.QueryRowContext(ctx,
		"SELECT NOT pg_is_in_recovery(), EXISTS (SELECT 1 FROM tasks)",
	).Scan(&writable, &tasksExist)
	if err != nil {
		return err
	}
	if !writable {
		return errors.New("database is in recovery")
	}
	return nil
}

func (s *Store) List(ctx context.Context) ([]Task, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, title, completed, created_at FROM tasks ORDER BY created_at DESC, id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]Task, 0)
	for rows.Next() {
		var task Task
		if err := rows.Scan(&task.ID, &task.Title, &task.Completed, &task.CreatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func (s *Store) Create(ctx context.Context, title string) (Task, error) {
	return readTask(s.db.QueryRowContext(ctx,
		"INSERT INTO tasks (title) VALUES ($1) RETURNING id, title, completed, created_at", title,
	))
}

func (s *Store) SetCompleted(ctx context.Context, id int64, completed bool) (Task, error) {
	return readTask(s.db.QueryRowContext(ctx,
		"UPDATE tasks SET completed = $2 WHERE id = $1 RETURNING id, title, completed, created_at", id, completed,
	))
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	var removed int64
	err := s.db.QueryRowContext(ctx, "DELETE FROM tasks WHERE id = $1 RETURNING id", id).Scan(&removed)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func readTask(row *sql.Row) (Task, error) {
	var task Task
	err := row.Scan(&task.ID, &task.Title, &task.Completed, &task.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	return task, err
}
