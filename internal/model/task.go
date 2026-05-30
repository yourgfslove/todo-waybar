package model

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Priority задаёт уровень важности задачи.
type Priority string

const (
	PriorityHigh Priority = "high"
	PriorityMid  Priority = "mid"
	PriorityLow  Priority = "low"
)

// Форматы дедлайна:
//   - DeadlineLayout      — только дата (хранится как есть)
//   - time.RFC3339        — дата + время (хранится в UTC)
// Ввод от пользователя может быть также "YYYY-MM-DD HH:MM" или длительностью.
const (
	DeadlineLayout     = "2006-01-02"
	DeadlineTimeLayout = "2006-01-02 15:04"
)

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

// MainTag возвращает главный тег задачи (первый в списке) — по нему задачи
// группируются. Пустая строка означает «без тегов».
func (t Task) MainTag() string {
	if len(t.Tags) > 0 {
		return t.Tags[0]
	}
	return ""
}

// SubTags возвращает подтеги (все теги, кроме главного).
func (t Task) SubTags() []string {
	if len(t.Tags) > 1 {
		return t.Tags[1:]
	}
	return nil
}

// SubTagsLabel форматирует подтеги как "#a #b" (пусто, если их нет).
func (t Task) SubTagsLabel() string {
	subs := t.SubTags()
	if len(subs) == 0 {
		return ""
	}
	parts := make([]string, len(subs))
	for i, s := range subs {
		parts[i] = "#" + s
	}
	return strings.Join(parts, " ")
}

// AllTagsLabel форматирует все теги как "#a #b" (пусто, если их нет) —
// для плоского режима, где нет заголовков групп.
func (t Task) AllTagsLabel() string {
	if len(t.Tags) == 0 {
		return ""
	}
	parts := make([]string, len(t.Tags))
	for i, s := range t.Tags {
		parts[i] = "#" + s
	}
	return strings.Join(parts, " ")
}

// DeadlineHasTime сообщает, что дедлайн хранит время (а не только дату).
func (t Task) DeadlineHasTime() bool {
	return strings.Contains(t.Deadline, "T")
}

// DeadlineTime парсит дедлайн в локальное время. ok=false, если дедлайна нет
// или он некорректен.
func (t Task) DeadlineTime() (time.Time, bool) {
	if t.Deadline == "" {
		return time.Time{}, false
	}
	if t.DeadlineHasTime() {
		if tm, err := time.Parse(time.RFC3339, t.Deadline); err == nil {
			return tm.Local(), true
		}
		return time.Time{}, false
	}
	if tm, err := time.ParseInLocation(DeadlineLayout, t.Deadline, time.Local); err == nil {
		return tm, true
	}
	return time.Time{}, false
}

// Overdue сообщает, что активная задача просрочена.
func (t Task) Overdue() bool {
	if t.Done {
		return false
	}
	d, ok := t.DeadlineTime()
	if !ok {
		return false
	}
	if t.DeadlineHasTime() {
		return d.Before(time.Now())
	}
	return d.Before(startOfDay(time.Now()))
}

// DueToday сообщает, что дедлайн активной задачи — сегодня (и ещё не прошёл).
func (t Task) DueToday() bool {
	if t.Done {
		return false
	}
	d, ok := t.DeadlineTime()
	if !ok {
		return false
	}
	now := time.Now()
	if t.DeadlineHasTime() {
		return sameDay(d, now) && !d.Before(now)
	}
	return startOfDay(d).Equal(startOfDay(now))
}

// DeadlineLabel возвращает человекочитаемый текст дедлайна (без стрелки/цвета):
//   - дата+время → "через 2ч 30м (30 May 18:45)" / "просрочено на 1ч (…)"
//   - только дата → "01 Jun" / "сегодня" / "просрочено (01 May)"
func (t Task) DeadlineLabel(now time.Time) string {
	d, ok := t.DeadlineTime()
	if !ok {
		return ""
	}
	if t.DeadlineHasTime() {
		return HumanRemaining(d.Sub(now)) + " (" + d.Format("02 Jan 15:04") + ")"
	}
	today := startOfDay(now)
	dd := startOfDay(d)
	switch {
	case dd.Before(today):
		return "просрочено (" + d.Format("02 Jan") + ")"
	case dd.Equal(today):
		return "сегодня"
	default:
		return d.Format("02 Jan")
	}
}

// DeadlineEditValue возвращает значение для предзаполнения поля редактирования.
func (t Task) DeadlineEditValue() string {
	if t.Deadline == "" {
		return ""
	}
	if t.DeadlineHasTime() {
		if d, ok := t.DeadlineTime(); ok {
			return d.Format(DeadlineTimeLayout)
		}
	}
	return t.Deadline
}

// HumanRemaining форматирует остаток времени до дедлайна (макс. 2 единицы).
func HumanRemaining(diff time.Duration) string {
	overdue := diff < 0
	if overdue {
		diff = -diff
	}
	var s string
	switch {
	case diff < time.Minute:
		s = "<1м"
	case diff < time.Hour:
		s = fmt.Sprintf("%dм", int(diff.Minutes()))
	case diff < 24*time.Hour:
		h := int(diff.Hours())
		m := int(diff.Minutes()) - h*60
		if m > 0 {
			s = fmt.Sprintf("%dч %dм", h, m)
		} else {
			s = fmt.Sprintf("%dч", h)
		}
	default:
		d := int(diff.Hours()) / 24
		h := int(diff.Hours()) % 24
		if h > 0 {
			s = fmt.Sprintf("%dд %dч", d, h)
		} else {
			s = fmt.Sprintf("%dд", d)
		}
	}
	if overdue {
		return "просрочено на " + s
	}
	return "через " + s
}

var durationRe = regexp.MustCompile(`(\d+)\s*(w|d|h|m)`)

// ParseDuration разбирает срок вида "2d3h30m", "45m", "1w", "3d".
// Единицы: w(недели) d(дни) h(часы) m(минуты), можно комбинировать.
func ParseDuration(s string) (time.Duration, error) {
	in := strings.ToLower(strings.TrimSpace(s))
	if in == "" {
		return 0, fmt.Errorf("пустой срок")
	}
	matches := durationRe.FindAllStringSubmatch(in, -1)
	if len(matches) == 0 {
		return 0, fmt.Errorf("не распознан срок %q", s)
	}
	// Убедимся, что вся строка состоит только из распознанных токенов.
	if leftover := strings.TrimSpace(durationRe.ReplaceAllString(in, "")); leftover != "" {
		return 0, fmt.Errorf("не распознан срок %q", s)
	}
	var total time.Duration
	for _, m := range matches {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			return 0, fmt.Errorf("не распознан срок %q", s)
		}
		switch m[2] {
		case "w":
			total += time.Duration(n) * 7 * 24 * time.Hour
		case "d":
			total += time.Duration(n) * 24 * time.Hour
		case "h":
			total += time.Duration(n) * time.Hour
		case "m":
			total += time.Duration(n) * time.Minute
		}
	}
	if total <= 0 {
		return 0, fmt.Errorf("срок должен быть положительным: %q", s)
	}
	return total, nil
}

// ParseDeadlineInput превращает пользовательский ввод в строку для хранения.
// Принимает: длительность (2d3h30m), "YYYY-MM-DD", "YYYY-MM-DD HH:MM", RFC3339.
// Пустой ввод → пустой дедлайн. now используется для относительных сроков.
func ParseDeadlineInput(s string, now time.Time) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	// Только дата — храним как есть.
	if _, err := time.ParseInLocation(DeadlineLayout, s, time.Local); err == nil {
		return s, nil
	}
	// Дата + время.
	if tm, err := time.ParseInLocation(DeadlineTimeLayout, s, time.Local); err == nil {
		return tm.UTC().Format(time.RFC3339), nil
	}
	// Уже RFC3339.
	if tm, err := time.Parse(time.RFC3339, s); err == nil {
		return tm.UTC().Format(time.RFC3339), nil
	}
	// Относительный срок.
	if dur, err := ParseDuration(s); err == nil {
		return now.Add(dur).UTC().Format(time.RFC3339), nil
	}
	return "", fmt.Errorf("не распознан срок/дата %q (примеры: 2d3h30m, 2026-06-01, \"2026-06-01 14:30\")", s)
}

func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
