package model

import "time"

// Priority задаёт уровень важности задачи.
type Priority string

const (
	PriorityHigh Priority = "high"
	PriorityMid  Priority = "mid"
	PriorityLow  Priority = "low"
)

// DeadlineLayout — формат хранения дедлайна (YYYY-MM-DD).
const DeadlineLayout = "2006-01-02"

// Task — единица todo-списка.
type Task struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	Done      bool      `json:"done"`
	Priority  Priority  `json:"priority"`
	Tags      []string  `json:"tags"`
	Deadline  string    `json:"deadline,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Valid сообщает, известен ли приоритет.
func (p Priority) Valid() bool {
	switch p {
	case PriorityHigh, PriorityMid, PriorityLow:
		return true
	}
	return false
}

// Rank возвращает порядок сортировки: high=0, mid=1, low=2.
// Неизвестный приоритет уходит в конец.
func (p Priority) Rank() int {
	switch p {
	case PriorityHigh:
		return 0
	case PriorityMid:
		return 1
	case PriorityLow:
		return 2
	default:
		return 3
	}
}

// Next возвращает следующий приоритет в цикле high → mid → low → high.
func (p Priority) Next() Priority {
	switch p {
	case PriorityHigh:
		return PriorityMid
	case PriorityMid:
		return PriorityLow
	default:
		return PriorityHigh
	}
}

// DeadlineTime парсит дедлайн в локальную дату. ok=false, если дедлайна нет
// или он некорректен.
func (t Task) DeadlineTime() (time.Time, bool) {
	if t.Deadline == "" {
		return time.Time{}, false
	}
	d, err := time.ParseInLocation(DeadlineLayout, t.Deadline, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return d, true
}

// Overdue сообщает, что активная задача просрочена (дедлайн раньше сегодня).
func (t Task) Overdue() bool {
	if t.Done {
		return false
	}
	d, ok := t.DeadlineTime()
	if !ok {
		return false
	}
	return d.Before(startOfDay(time.Now()))
}

// DueToday сообщает, что дедлайн активной задачи — сегодня.
func (t Task) DueToday() bool {
	if t.Done {
		return false
	}
	d, ok := t.DeadlineTime()
	if !ok {
		return false
	}
	return d.Equal(startOfDay(time.Now()))
}

func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
