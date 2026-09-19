package todo

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type testStore struct {
	task Task
	err  error
}

func (s *testStore) Ready(context.Context) error { return s.err }

func (s *testStore) List(context.Context) ([]Task, error) {
	if s.task.ID == 0 {
		return []Task{}, s.err
	}
	return []Task{s.task}, s.err
}

func (s *testStore) Create(_ context.Context, title string) (Task, error) {
	s.task = Task{ID: 1, Title: title}
	return s.task, s.err
}

func (s *testStore) SetCompleted(_ context.Context, id int64, completed bool) (Task, error) {
	if id != s.task.ID {
		return Task{}, ErrNotFound
	}
	s.task.Completed = completed
	return s.task, s.err
}

func (s *testStore) Delete(_ context.Context, id int64) error {
	if id != s.task.ID {
		return ErrNotFound
	}
	s.task = Task{}
	return s.err
}

func request(handler http.Handler, method, path, contentType, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestTaskLifecycle(t *testing.T) {
	h := NewHandler(&testStore{}, "test-version", slog.New(slog.NewTextHandler(io.Discard, nil)))
	steps := []struct {
		method, path, body string
		status             int
	}{
		{"GET", "/api/tasks", "", 200},
		{"POST", "/api/tasks", `{"title":"  Контрольная задача  "}`, 201},
		{"PATCH", "/api/tasks/1", `{"completed":true}`, 200},
		{"PATCH", "/api/tasks/1", `{"completed":false}`, 200},
		{"DELETE", "/api/tasks/1", "", 204},
		{"DELETE", "/api/tasks/1", "", 404},
	}
	for i, step := range steps {
		rec := request(h, step.method, step.path, "application/json", step.body)
		if rec.Code != step.status || rec.Header().Get("X-App-Version") != "test-version" {
			t.Fatalf("step %d: got HTTP %d, body %q", i, rec.Code, rec.Body.String())
		}
		if i == 1 || i == 2 || i == 3 {
			var task Task
			if err := json.Unmarshal(rec.Body.Bytes(), &task); err != nil {
				t.Fatal(err)
			}
			if task.ID != 1 || task.Title != "Контрольная задача" || task.Completed != (i == 2) {
				t.Fatalf("step %d: unexpected task %+v", i, task)
			}
		}
	}
}

func TestInputValidation(t *testing.T) {
	h := NewHandler(&testStore{}, "dev", slog.New(slog.NewTextHandler(io.Discard, nil)))
	tests := []struct {
		name, method, path, media, body string
		status                          int
	}{
		{"missing title", "POST", "/api/tasks", "application/json", "{}", 400},
		{"extra value", "POST", "/api/tasks", "application/json", `{"title":"a"} {}`, 400},
		{"unknown key", "POST", "/api/tasks", "application/json", `{"title":"a","done":true}`, 400},
		{"invalid media", "POST", "/api/tasks", "text/plain", `{"title":"a"}`, 415},
		{"large body", "POST", "/api/tasks", "application/json", `{"title":"` + strings.Repeat("x", 8192) + `"}`, 413},
		{"invalid id", "DELETE", "/api/tasks/-1", "", "", 400},
		{"missing status", "PATCH", "/api/tasks/1", "application/json", "{}", 400},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := request(h, tc.method, tc.path, tc.media, tc.body)
			if rec.Code != tc.status {
				t.Errorf("got HTTP %d, want %d: %s", rec.Code, tc.status, rec.Body)
			}
		})
	}
}

func TestDatabaseOutage(t *testing.T) {
	h := NewHandler(&testStore{err: errors.New("test connection refused")}, "dev",
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	for path, status := range map[string]int{"/livez": 200, "/readyz": 503, "/api/tasks": 503} {
		rec := request(h, "GET", path, "", "")
		if rec.Code != status || strings.Contains(rec.Body.String(), "connection refused") {
			t.Errorf("%s: got HTTP %d, body %q", path, rec.Code, rec.Body)
		}
	}
}
