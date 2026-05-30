package ui

import (
	"strings"

	"gtodo/internal/model"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type formField int

const (
	fieldText formField = iota
	fieldPriority
	fieldTags
	fieldDeadline
	fieldCount
)

// formAction — результат обработки клавиши формой.
type formAction int

const (
	actionNone formAction = iota
	actionSubmit
	actionCancel
)

// form — экран добавления/редактирования задачи.
type form struct {
	title     string
	editingID string // пусто => режим добавления
	focus     formField
	text      textinput.Model
	tags      textinput.Model
	deadline  textinput.Model
	priority  model.Priority
}

func newForm(task *model.Task) form {
	mk := func(placeholder string, limit int) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.CharLimit = limit
		ti.Prompt = ""
		return ti
	}

	f := form{
		title:    "Новая задача",
		focus:    fieldText,
		text:     mk("Текст задачи", 200),
		tags:     mk("work, home", 100),
		deadline: mk("2d3h30m или YYYY-MM-DD", 30),
		priority: model.PriorityMid,
	}

	if task != nil {
		f.title = "Редактирование"
		f.editingID = task.ID
		f.text.SetValue(task.Text)
		f.tags.SetValue(strings.Join(task.Tags, ", "))
		f.deadline.SetValue(task.DeadlineEditValue())
		if task.Priority.Valid() {
			f.priority = task.Priority
		}
	}

	f.applyFocus()
	return f
}

// applyFocus фокусирует активное текстовое поле и снимает фокус с прочих.
func (f *form) applyFocus() tea.Cmd {
	f.text.Blur()
	f.tags.Blur()
	f.deadline.Blur()
	switch f.focus {
	case fieldText:
		return f.text.Focus()
	case fieldTags:
		return f.tags.Focus()
	case fieldDeadline:
		return f.deadline.Focus()
	}
	return nil
}

func (f *form) next() tea.Cmd {
	f.focus = (f.focus + 1) % fieldCount
	return f.applyFocus()
}

func (f *form) prev() tea.Cmd {
	f.focus = (f.focus - 1 + fieldCount) % fieldCount
	return f.applyFocus()
}

func (f form) update(msg tea.Msg) (form, formAction, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		// прокидываем прочие сообщения в активный input
		return f.propagate(msg)
	}

	switch key.String() {
	case "esc":
		return f, actionCancel, nil
	case "tab", "down":
		return f, actionNone, f.next()
	case "shift+tab", "up":
		return f, actionNone, f.prev()
	case "enter":
		if f.focus == fieldDeadline {
			return f, actionSubmit, nil
		}
		return f, actionNone, f.next()
	}

	// Поле приоритета — не textinput: листаем значения.
	if f.focus == fieldPriority {
		switch key.String() {
		case "left", "h", "shift+left":
			f.priority = prevPriority(f.priority)
			return f, actionNone, nil
		case "right", "l", " ", "shift+right":
			f.priority = f.priority.Next()
			return f, actionNone, nil
		}
		return f, actionNone, nil
	}

	return f.propagate(msg)
}

func (f form) propagate(msg tea.Msg) (form, formAction, tea.Cmd) {
	var cmd tea.Cmd
	switch f.focus {
	case fieldText:
		f.text, cmd = f.text.Update(msg)
	case fieldTags:
		f.tags, cmd = f.tags.Update(msg)
	case fieldDeadline:
		f.deadline, cmd = f.deadline.Update(msg)
	}
	return f, actionNone, cmd
}

// values возвращает введённые данные в нормализованном виде.
func (f form) values() (text string, priority model.Priority, tags []string, deadline string) {
	text = strings.TrimSpace(f.text.Value())
	priority = f.priority
	for _, t := range strings.Split(f.tags.Value(), ",") {
		if t = strings.TrimSpace(t); t != "" {
			tags = append(tags, t)
		}
	}
	deadline = strings.TrimSpace(f.deadline.Value())
	return
}

func (f form) view() string {
	row := func(field formField, label, value string) string {
		marker := "  "
		lbl := formLabelStyle
		if f.focus == field {
			marker = cursorStyle.Render("▌ ")
			lbl = lbl.Foreground(colAccent)
		}
		return marker + lbl.Render(label) + value
	}

	prioVal := lipgloss.NewStyle().
		Foreground(priorityColor(f.priority)).
		Render(priorityBadge(f.priority) + " " + string(f.priority))
	if f.focus == fieldPriority {
		prioVal += helpStyle.Render("  (← → сменить)")
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(f.title) + "\n\n")
	b.WriteString(row(fieldText, "Текст", f.text.View()) + "\n")
	b.WriteString(row(fieldPriority, "Приоритет", prioVal) + "\n")
	b.WriteString(row(fieldTags, "Теги", f.tags.View()) + "\n")
	b.WriteString(row(fieldDeadline, "Дедлайн", f.deadline.View()) + "\n\n")
	b.WriteString(helpStyle.Render("Tab/↑↓ — поля • Enter — далее/сохранить • Esc — отмена"))

	return formBoxStyle.Render(b.String())
}

func prevPriority(p model.Priority) model.Priority {
	// обратный цикл к Next(): high → low → mid → high
	switch p {
	case model.PriorityHigh:
		return model.PriorityLow
	case model.PriorityLow:
		return model.PriorityMid
	default:
		return model.PriorityHigh
	}
}
