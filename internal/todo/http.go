package todo

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const operationTimeout = 2 * time.Second

type taskStore interface {
	Ready(context.Context) error
	List(context.Context) ([]Task, error)
	Create(context.Context, string) (Task, error)
	SetCompleted(context.Context, int64, bool) (Task, error)
	Delete(context.Context, int64) error
}

type api struct {
	store  taskStore
	logger *slog.Logger
}

func NewHandler(store taskStore, version string, logger *slog.Logger) http.Handler {
	a := api{store: store, logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /livez", func(w http.ResponseWriter, _ *http.Request) {
		reply(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", a.ready)
	mux.HandleFunc("GET /api/tasks", a.list)
	mux.HandleFunc("POST /api/tasks", a.create)
	mux.HandleFunc("PATCH /api/tasks/{id}", a.setCompleted)
	mux.HandleFunc("DELETE /api/tasks/{id}", a.delete)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-App-Version", version)
		w.Header().Set("Cache-Control", "no-store")
		mux.ServeHTTP(w, r)
	})
}

func (a api) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), operationTimeout)
	defer cancel()
	if err := a.store.Ready(ctx); err != nil {
		reply(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
		return
	}
	reply(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (a api) list(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), operationTimeout)
	defer cancel()
	tasks, err := a.store.List(ctx)
	if err != nil {
		a.storeError(w, "list", err)
		return
	}
	reply(w, http.StatusOK, tasks)
}

func (a api) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string `json:"title"`
	}
	if !decode(w, r, &body) {
		return
	}
	title := strings.TrimSpace(body.Title)
	if size := utf8.RuneCountInString(title); size < 1 || size > 160 || strings.ContainsRune(title, 0) {
		problem(w, http.StatusBadRequest, "title must contain 1 to 160 characters and no NUL")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), operationTimeout)
	defer cancel()
	task, err := a.store.Create(ctx, title)
	if err != nil {
		a.storeError(w, "create", err)
		return
	}
	reply(w, http.StatusCreated, task)
}

func (a api) setCompleted(w http.ResponseWriter, r *http.Request) {
	id, ok := taskID(w, r)
	if !ok {
		return
	}
	var body struct {
		Completed *bool `json:"completed"`
	}
	if !decode(w, r, &body) {
		return
	}
	if body.Completed == nil {
		problem(w, http.StatusBadRequest, "completed must be a boolean")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), operationTimeout)
	defer cancel()
	task, err := a.store.SetCompleted(ctx, id, *body.Completed)
	if err != nil {
		a.storeError(w, "set completed", err)
		return
	}
	reply(w, http.StatusOK, task)
}

func (a api) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := taskID(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), operationTimeout)
	defer cancel()
	if err := a.store.Delete(ctx, id); err != nil {
		a.storeError(w, "delete", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a api) storeError(w http.ResponseWriter, operation string, err error) {
	if errors.Is(err, ErrNotFound) {
		problem(w, http.StatusNotFound, "task not found")
		return
	}
	a.logger.Error("database operation failed", "operation", operation, "error", err)
	problem(w, http.StatusServiceUnavailable, "database unavailable")
}

func taskID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		problem(w, http.StatusBadRequest, "invalid task id")
		return 0, false
	}
	return id, true
}

func decode(w http.ResponseWriter, r *http.Request, value any) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		problem(w, http.StatusUnsupportedMediaType, "expected application/json")
		return false
	}

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
	decoder.DisallowUnknownFields()
	err = decoder.Decode(value)
	if err == nil {
		err = decoder.Decode(new(json.RawMessage))
		if err == io.EOF {
			return true
		}
	}

	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		problem(w, http.StatusRequestEntityTooLarge, "request body exceeds 8 KiB")
		return false
	}
	problem(w, http.StatusBadRequest, "expected one valid JSON object")
	return false
}

func problem(w http.ResponseWriter, status int, message string) {
	reply(w, status, map[string]string{"error": message})
}

func reply(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
