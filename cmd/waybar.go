package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"gtodo/internal/model"
	"gtodo/internal/store"

	"github.com/spf13/cobra"
)

const waybarTooltipMax = 10

// waybarSignal — номер RT-сигнала модуля custom/gtodo (waybar: "signal": N).
const waybarSignal = 10

// notifyWaybar мгновенно обновляет модуль waybar после изменения задач.
// Ошибки игнорируются: waybar может быть не запущен или собран без модуля.
func notifyWaybar() {
	_ = exec.Command("pkill", "-RTMIN+"+strconv.Itoa(waybarSignal), "waybar").Run()
}

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

	urgent := false
	for _, t := range active {
		if t.Overdue() || t.Priority == model.PriorityHigh {
			urgent = true
			break
		}
	}

	out := waybarOutput{
		Text:    fmt.Sprintf("✓ %d  %d", done, len(tasks)),
		Tooltip: groupedTooltip(active),
	}
	if urgent {
		out.Class = "has-urgent"
	}
	return out
}

// groupedTooltip строит tooltip: активные задачи, сгруппированные по главному
// тегу (по алфавиту, «без тегов» в конце), внутри группы — по приоритету.
// Всего показывается не более waybarTooltipMax задач.
func groupedTooltip(active []model.Task) string {
	buckets := map[string][]model.Task{}
	var keys []string
	for _, t := range active {
		k := t.MainTag()
		if _, ok := buckets[k]; !ok {
			keys = append(keys, k)
		}
		buckets[k] = append(buckets[k], t)
	}
	sort.Slice(keys, func(i, j int) bool {
		if (keys[i] == "") != (keys[j] == "") {
			return keys[j] == ""
		}
		return strings.ToLower(keys[i]) < strings.ToLower(keys[j])
	})

	var lines []string
	shown := 0
	for _, k := range keys {
		if shown >= waybarTooltipMax {
			break
		}
		ts := buckets[k]
		store.Sort(ts, store.SortPriority)

		name := k
		if name == "" {
			name = "без тегов"
		}
		var group []string
		for _, t := range ts {
			if shown >= waybarTooltipMax {
				break
			}
			group = append(group, "  "+waybarLine(t))
			shown++
		}
		if len(group) > 0 {
			lines = append(lines, name)
			lines = append(lines, group...)
		}
	}
	return strings.Join(lines, "\n")
}

func waybarLine(t model.Task) string {
	var b strings.Builder
	b.WriteString("● ")
	if sub := t.SubTagsLabel(); sub != "" {
		b.WriteString(sub)
		b.WriteString(" ")
	}
	b.WriteString(t.Text)
	b.WriteString(" [")
	b.WriteString(string(t.Priority))
	b.WriteString("]")
	if label := t.DeadlineLabel(time.Now()); label != "" {
		b.WriteString(" → ")
		b.WriteString(label)
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
