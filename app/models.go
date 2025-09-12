package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type TaskFormResultMsg struct {
	Result bool
	Title  string
	Notes  string
}

type DaySelectedMsg struct {
	Day time.Time
}

// Task represents a single to-do item.
type Task struct {
	ID          int        `json:"id"`
	Date        time.Time  `json:"date"`
	Title       string     `json:"title"`
	Notes       string     `json:"notes,omitempty"`
	Due         *time.Time `json:"due,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// IsCompleted returns true if the task is completed.
func (t *Task) IsCompleted() bool {
	return t.CompletedAt != nil
}

// Store manages tasks persisted to a JSON file.
// It is safe for concurrent use.
type Store struct {
	mu     sync.RWMutex
	path   string
	tasks  []*Task
	NextID int
}

// Errors returned by Store operations.
var (
	ErrNotFound      = errors.New("task not found")
	ErrTitleRequired = errors.New("title is required")
)

// Load opens (or initializes) a task store backed by the given JSON file.
// If the file does not exist, an empty store is created on first Save.
func Load(path string) (*Store, error) {
	s := &Store{path: path, tasks: []*Task{}, NextID: 1}

	f, err := os.Open(path)
	if err != nil {
		// If file doesn't exist yet, that's fine—start empty.
		if errors.Is(err, os.ErrNotExist) {
			return s, nil
		}
		return nil, fmt.Errorf("open store: %w", err)
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()

	var onDisk struct {
		Tasks []*Task `json:"tasks"`
	}
	if err := dec.Decode(&onDisk); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("decode store: %w", err)
	}
	s.tasks = onDisk.Tasks
	// Determine nextID from max existing ID.
	maxID := 0
	for _, t := range s.tasks {
		if t.ID > maxID {
			maxID = t.ID
		}
	}
	s.NextID = maxID + 1
	return s, nil
}

// Save writes the current tasks to disk atomically.
func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir store dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".tasks-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()

	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	payload := struct {
		Tasks []*Task `json:"tasks"`
	}{Tasks: s.tasks}

	if err := enc.Encode(payload); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("encode store: %w", err)
	}
	// Ensure data hits disk before rename.
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("sync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp: %w", err)
	}
	// Atomic replace.
	if err := os.Rename(tmpPath, s.path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename store: %w", err)
	}
	// Best effort directory sync (ignore errors on some OSes).
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}

// List returns a copy of tasks, sorted by:
// 1) incomplete first by due date (nil due goes last),
// 2) then completed by completion time,
// 3) finally by ID for stability.
func (s *Store) List() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*Task, len(s.tasks))
	copy(out, s.tasks)

	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]

		// Incomplete before complete
		if a.IsCompleted() != b.IsCompleted() {
			return !a.IsCompleted()
		}

		// If both incomplete: sort by due date (nil last), then ID
		if !a.IsCompleted() {
			if a.Due == nil && b.Due != nil {
				return false
			}
			if a.Due != nil && b.Due == nil {
				return true
			}
			if a.Due != nil && b.Due != nil && !a.Due.Equal(*b.Due) {
				return a.Due.Before(*b.Due)
			}
			return a.ID < b.ID
		}

		// If both complete: newest completion first
		if a.CompletedAt != nil && b.CompletedAt != nil && !a.CompletedAt.Equal(*b.CompletedAt) {
			return b.CompletedAt.Before(*a.CompletedAt)
		}
		return a.ID < b.ID
	})
	return out
}

func (s *Store) ListByDate(date time.Time) []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []*Task
	for _, t := range s.tasks {
		if t.Date.Year() == date.Year() && t.Date.Month() == date.Month() && t.Date.Day() == date.Day() {
			out = append(out, t)
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]

		// Incomplete before complete
		if a.IsCompleted() != b.IsCompleted() {
			return !a.IsCompleted()
		}

		// If both incomplete: sort by due date (nil last), then ID
		if !a.IsCompleted() {
			if a.Due == nil && b.Due != nil {
				return false
			}
			if a.Due != nil && b.Due == nil {
				return true
			}
			if a.Due != nil && b.Due != nil && !a.Due.Equal(*b.Due) {
				return a.Due.Before(*b.Due)
			}
			return a.ID < b.ID
		}

		// If both complete: newest completion first
		if a.CompletedAt != nil && b.CompletedAt != nil && !a.CompletedAt.Equal(*b.CompletedAt) {
			return b.CompletedAt.Before(*a.CompletedAt)
		}
		return a.ID < b.ID
	})
	return out
}

// Add creates a new task and saves it to disk.
func (s *Store) Add(title, notes string, due *time.Time, date time.Time) (*Task, error) {
	if title == "" {
		return nil, ErrTitleRequired
	}

	now := time.Now()
	s.mu.Lock()
	t := &Task{
		ID:        s.NextID,
		Date:      date,
		Title:     title,
		Notes:     notes,
		Due:       cloneTimePtr(due),
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.tasks = append(s.tasks, t)
	s.NextID++
	s.mu.Unlock()

	if err := s.Save(); err != nil {
		return nil, err
	}
	return t, nil
}

// Get returns a copy of the task with the given ID.
func (s *Store) Get(id int) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, t := range s.tasks {
		if t.ID == id {
			cp := *t
			cp.Due = cloneTimePtr(t.Due)
			cp.CompletedAt = cloneTimePtr(t.CompletedAt)
			return &cp, nil
		}
	}
	return nil, ErrNotFound
}

// UpdateOptions defines which fields to change.
// Use pointer fields so "nil" means "leave unchanged".
type UpdateOptions struct {
	Title *string
	Notes *string
	Due   **time.Time // pointer to a *time.Time: nil=leave, &nil=clear, &time=update
}

// Update modifies a task and saves it.
func (s *Store) Update(id int, opts UpdateOptions) (*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := s.findUnsafe(id)
	if t == nil {
		return nil, ErrNotFound
	}

	if opts.Title != nil {
		if *opts.Title == "" {
			return nil, ErrTitleRequired
		}
		t.Title = *opts.Title
	}
	if opts.Notes != nil {
		t.Notes = *opts.Notes
	}
	if opts.Due != nil {
		// *opts.Due may be nil (clear) or &time (set)
		if *opts.Due == nil {
			t.Due = nil
		} else {
			t.Due = cloneTimePtr(*opts.Due)
		}
	}
	t.UpdatedAt = time.Now()

	// Persist outside the lock boundary to reduce contention,
	// but we already mutated; to keep simple, save while still holding lock.
	// (If your workload is heavy, copy tasks and save without lock.)
	if err := s.saveUnsafe(); err != nil {
		return nil, err
	}
	cp := *t
	cp.Due = cloneTimePtr(t.Due)
	cp.CompletedAt = cloneTimePtr(t.CompletedAt)
	return &cp, nil
}

// MarkCompleted sets or clears completion and saves it.
func (s *Store) MarkCompleted(id int, completed bool) (*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := s.findUnsafe(id)
	if t == nil {
		return nil, ErrNotFound
	}

	now := time.Now()
	if completed {
		if t.CompletedAt == nil {
			t.CompletedAt = &now
		}
	} else {
		t.CompletedAt = nil
	}
	t.UpdatedAt = now

	if err := s.saveUnsafe(); err != nil {
		return nil, err
	}
	cp := *t
	cp.Due = cloneTimePtr(t.Due)
	cp.CompletedAt = cloneTimePtr(t.CompletedAt)
	return &cp, nil
}

// ToggleCompleted flips completion state and saves it.
func (s *Store) ToggleCompleted(id int) (*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := s.findUnsafe(id)
	if t == nil {
		return nil, ErrNotFound
	}

	now := time.Now()
	if t.CompletedAt == nil {
		t.CompletedAt = &now
	} else {
		t.CompletedAt = nil
	}
	t.UpdatedAt = now

	if err := s.saveUnsafe(); err != nil {
		return nil, err
	}
	cp := *t
	cp.Due = cloneTimePtr(t.Due)
	cp.CompletedAt = cloneTimePtr(t.CompletedAt)
	return &cp, nil
}

// Delete removes a task and saves it.
func (s *Store) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i, t := range s.tasks {
		if t.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return ErrNotFound
	}

	// Remove without preserving order (but here we preserve for readability).
	s.tasks = append(s.tasks[:idx], s.tasks[idx+1:]...)
	return s.saveUnsafe()
}

// --- helpers ---

func (s *Store) findUnsafe(id int) *Task {
	for _, t := range s.tasks {
		if t.ID == id {
			return t
		}
	}
	return nil
}

func (s *Store) saveUnsafe() error {
	// We are holding the write lock; call the public Save via a thin wrapper
	// that doesn't attempt to lock again.
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir store dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".tasks-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()

	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	payload := struct {
		Tasks []*Task `json:"tasks"`
	}{Tasks: s.tasks}

	if err := enc.Encode(payload); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("encode store: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("sync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename store: %w", err)
	}
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}

func cloneTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	cp := *t
	return &cp
}

func NextDay(t time.Time) tea.Cmd {
	return func() tea.Msg {
		return DaySelectedMsg{
			Day: t.AddDate(0, 0, 1),
		}
	}
}

func PrevDay(t time.Time) tea.Cmd {
	return func() tea.Msg {
		return DaySelectedMsg{
			Day: t.AddDate(0, 0, -1),
		}
	}
}

func Today() tea.Cmd {
	return func() tea.Msg {
		return DaySelectedMsg{
			Day: time.Now(),
		}
	}
}
