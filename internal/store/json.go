package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gtodo/internal/model"
)

// ErrNotFound возвращается, когда задача с указанным id отсутствует.
var ErrNotFound = errors.New("task not found")

// Store читает и пишет задачи в JSON-файл.
type Store struct {
	path string
}

// New создаёт Store с путём по умолчанию:
// $XDG_DATA_HOME/gtodo/tasks.json или ~/.local/share/gtodo/tasks.json.
func New() (*Store, error) {
	p, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return &Store{path: p}, nil
}

// NewWithPath создаёт Store с явным путём (удобно для тестов).
func NewWithPath(path string) *Store {
	return &Store{path: path}
}

// DefaultPath вычисляет стандартный путь к файлу задач.
func DefaultPath() (string, error) {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, "gtodo", "tasks.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "gtodo", "tasks.json"), nil
}

// Path возвращает путь к файлу задач.
func (s *Store) Path() string { return s.path }

// Load читает задачи. Отсутствующий файл — это пустой список, а не ошибка.
func (s *Store) Load() ([]model.Task, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []model.Task{}, nil
		}
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return []model.Task{}, nil
	}
	var tasks []model.Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// Save атомарно записывает задачи (через временный файл + rename).
func (s *Store) Save(tasks []model.Task) error {
	if tasks == nil {
		tasks = []model.Task{}
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".tasks-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // на случай ошибки до rename

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, s.path)
}

// Find ищет задачу по id или по уникальному префиксу id.
// Возвращает индекс в срезе.
func Find(tasks []model.Task, id string) (int, error) {
	// Точное совпадение.
	for i := range tasks {
		if tasks[i].ID == id {
			return i, nil
		}
	}
	// Префикс id (удобно в CLI).
	match := -1
	for i := range tasks {
		if strings.HasPrefix(tasks[i].ID, id) {
			if match != -1 {
				return -1, errors.New("ambiguous id prefix: " + id)
			}
			match = i
		}
	}
	if match == -1 {
		return -1, ErrNotFound
	}
	return match, nil
}

// SortMode задаёт порядок сортировки.
type SortMode int

const (
	SortPriority SortMode = iota // по приоритету
	SortDeadline                 // по дедлайну
	SortCreated                  // по дате создания
)

func (m SortMode) String() string {
	switch m {
	case SortPriority:
		return "priority"
	case SortDeadline:
		return "deadline"
	case SortCreated:
		return "created"
	default:
		return "priority"
	}
}

// Next возвращает следующий режим сортировки по циклу.
func (m SortMode) Next() SortMode {
	return (m + 1) % 3
}

// Sort сортирует срез задач на месте по выбранному режиму.
// Внутри одинаковых ключей — стабильный порядок: невыполненные раньше
// выполненных, затем по дате создания.
func Sort(tasks []model.Task, mode SortMode) {
	sort.SliceStable(tasks, func(i, j int) bool {
		a, b := tasks[i], tasks[j]
		switch mode {
		case SortDeadline:
			da, oka := a.DeadlineTime()
			db, okb := b.DeadlineTime()
			if oka != okb {
				return oka // задачи с дедлайном раньше
			}
			if oka && okb && !da.Equal(db) {
				return da.Before(db)
			}
		case SortCreated:
			if !a.CreatedAt.Equal(b.CreatedAt) {
				return a.CreatedAt.Before(b.CreatedAt)
			}
		default: // SortPriority
			if a.Priority.Rank() != b.Priority.Rank() {
				return a.Priority.Rank() < b.Priority.Rank()
			}
		}
		// Тай-брейк.
		if a.Done != b.Done {
			return !a.Done
		}
		return a.CreatedAt.Before(b.CreatedAt)
	})
}
