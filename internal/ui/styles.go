package ui

import (
	"gtodo/internal/model"

	"github.com/charmbracelet/lipgloss"
)

// Палитра (Dracula-подобная, хорошо ложится на тёмные темы Omarchy).
var (
	colHigh    = lipgloss.Color("#ff5555")
	colMid     = lipgloss.Color("#f1fa8c")
	colLow     = lipgloss.Color("#50fa7b")
	colSubtle  = lipgloss.Color("#6272a4")
	colAccent  = lipgloss.Color("#bd93f9")
	colFg      = lipgloss.Color("#f8f8f2")
	colOverdue = lipgloss.Color("#ff5555")
	colToday   = lipgloss.Color("#f1fa8c")
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colFg).
			Background(colAccent).
			Padding(0, 1)

	filterStyle = lipgloss.NewStyle().Foreground(colAccent)

	headerStyle = lipgloss.NewStyle().
			Foreground(colSubtle).
			Padding(0, 1)

	cursorStyle = lipgloss.NewStyle().Foreground(colAccent).Bold(true)

	selectedLineStyle = lipgloss.NewStyle().Foreground(colFg).Bold(true)
	normalLineStyle   = lipgloss.NewStyle().Foreground(colFg)
	doneLineStyle     = lipgloss.NewStyle().Foreground(colSubtle).Strikethrough(true)

	tagStyle = lipgloss.NewStyle().Foreground(colAccent)

	groupHeaderStyle = lipgloss.NewStyle().
				Foreground(colAccent).
				Bold(true).
				Underline(true)

	groupCountStyle = lipgloss.NewStyle().Foreground(colSubtle)

	helpStyle = lipgloss.NewStyle().Foreground(colSubtle).Padding(1, 1, 0, 1)

	statusStyle = lipgloss.NewStyle().Foreground(colMid).Padding(0, 1)

	formLabelStyle = lipgloss.NewStyle().Foreground(colSubtle).Width(10)
	formBoxStyle   = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colAccent).
			Padding(1, 2)
)

// priorityBadge возвращает цветной кружок-бейдж приоритета.
func priorityBadge(p model.Priority) string {
	switch p {
	case model.PriorityHigh:
		return lipgloss.NewStyle().Foreground(colHigh).Render("🔴")
	case model.PriorityMid:
		return lipgloss.NewStyle().Foreground(colMid).Render("🟡")
	case model.PriorityLow:
		return lipgloss.NewStyle().Foreground(colLow).Render("🟢")
	default:
		return " "
	}
}

func priorityColor(p model.Priority) lipgloss.Color {
	switch p {
	case model.PriorityHigh:
		return colHigh
	case model.PriorityMid:
		return colMid
	default:
		return colLow
	}
}
