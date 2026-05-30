package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gtodo/internal/model"
	"gtodo/internal/store"

	"github.com/spf13/cobra"
)

const waybarTooltipMax = 10

// waybarOutput — формат, который ожидает Waybar при return-type: json.
type waybarOutput struct {
	Text    string `json:"text"`
	Tooltip string `json:"tooltip"`
	Class   string `json:"class"`
}

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "waybar",
		Short: "Вывести JSON для модуля Waybar",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			s := mustStore()
			tasks, err := s.Load()
			if err != nil {
				// Не роняем бар: показываем пустое состояние.
				return printWaybar(waybarOutput{Text: "✓ 0  0"})
			}
			return printWaybar(buildWaybar(tasks))
		},
	})
}

func buildWaybar(tasks []model.Task) waybarOutput {
	done := 0
	for _, t := range tasks {
		if t.Done {
			done++
		}
	}

	active := filterFunc(tasks, func(t model.Task) bool { return !t.Done })
	store.Sort(active, store.SortPriority)

	urgent := false
	var lines []string
	for _, t := range active {
		if t.Overdue() || t.Priority == model.PriorityHigh {
			urgent = true
		}
		if len(lines) < waybarTooltipMax {
			lines = append(lines, waybarLine(t))
		}
	}

	out := waybarOutput{
		Text:    fmt.Sprintf("✓ %d  %d", done, len(tasks)),
		Tooltip: strings.Join(lines, "\n"),
		Class:   "",
	}
	if urgent {
		out.Class = "has-urgent"
	}
	return out
}

func waybarLine(t model.Task) string {
	var b strings.Builder
	b.WriteString("● ")
	b.WriteString(t.Text)
	b.WriteString(" [")
	b.WriteString(string(t.Priority))
	b.WriteString("]")
	if d, ok := t.DeadlineTime(); ok {
		b.WriteString(" → ")
		switch {
		case t.Overdue():
			b.WriteString("просрочено (" + d.Format("02 Jan") + ")")
		case t.DueToday():
			b.WriteString("сегодня")
		default:
			b.WriteString(d.Format("02 Jan"))
		}
	}
	return b.String()
}

func printWaybar(o waybarOutput) error {
	data, err := json.Marshal(o)
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, string(data))
	return nil
}
