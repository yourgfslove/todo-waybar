package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"gtodo/internal/model"
	"gtodo/internal/store"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(addCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(doneCmd())
	rootCmd.AddCommand(undoneCmd())
	rootCmd.AddCommand(rmCmd())
	rootCmd.AddCommand(getCmd())
}

func addCmd() *cobra.Command {
	var (
		priority string
		tags     []string
		deadline string
		in       string
	)
	cmd := &cobra.Command{
		Use:   "add <text>",
		Short: "Добавить задачу",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p := model.Priority(priority)
			if !p.Valid() {
				return fmt.Errorf("недопустимый приоритет %q (ожидается high/mid/low)", priority)
			}
			if deadline != "" && in != "" {
				return fmt.Errorf("укажите либо --deadline, либо --in, но не оба")
			}

			now := time.Now()
			stored := ""
			switch {
			case in != "":
				dur, err := model.ParseDuration(in)
				if err != nil {
					return fmt.Errorf("неверный срок --in: %w", err)
				}
				stored = now.Add(dur).UTC().Format(time.RFC3339)
			case deadline != "":
				var err error
				stored, err = model.ParseDeadlineInput(deadline, now)
				if err != nil {
					return err
				}
			}

			s := mustStore()
			tasks, err := s.Load()
			if err != nil {
				return err
			}
			task := model.Task{
				ID:        uuid.NewString(),
				Text:      args[0],
				Priority:  p,
				Tags:      normalizeTags(tags),
				Deadline:  stored,
				CreatedAt: now.UTC(),
			}
			tasks = append(tasks, task)
			if err := s.Save(tasks); err != nil {
				return err
			}
			notifyWaybar()
			fmt.Println(task.ID)
			return nil
		},
	}
	cmd.Flags().StringVarP(&priority, "priority", "p", string(model.PriorityMid), "приоритет: high/mid/low")
	cmd.Flags().StringSliceVarP(&tags, "tags", "t", nil, "теги через запятую")
	cmd.Flags().StringVarP(&deadline, "deadline", "d", "", "дедлайн: YYYY-MM-DD или \"YYYY-MM-DD HH:MM\"")
	cmd.Flags().StringVarP(&in, "in", "i", "", "срок от создания: напр. 2d3h30m, 45m, 1w")
	return cmd
}

func listCmd() *cobra.Command {
	var (
		filter   string
		sortFlag string
		asJSON   bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Показать задачи",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			s := mustStore()
			tasks, err := s.Load()
			if err != nil {
				return err
			}
			tasks = applyFilter(tasks, filter)
			store.Sort(tasks, parseSort(sortFlag))

			if asJSON {
				return printJSON(tasks)
			}
			if len(tasks) == 0 {
				fmt.Println("задач нет")
				return nil
			}
			for _, t := range tasks {
				fmt.Println(formatTaskLine(t))
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&filter, "filter", "f", "all", "фильтр: all/active/done")
	cmd.Flags().StringVarP(&sortFlag, "sort", "s", "priority", "сортировка: priority/deadline/created")
	cmd.Flags().BoolVar(&asJSON, "json", false, "вывести JSON")
	return cmd
}

func doneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "done <id>",
		Short: "Отметить задачу выполненной",
		Args:  cobra.ExactArgs(1),
		RunE:  func(_ *cobra.Command, args []string) error { return setDone(args[0], true) },
	}
}

func undoneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "undone <id>",
		Short: "Снять отметку выполнения",
		Args:  cobra.ExactArgs(1),
		RunE:  func(_ *cobra.Command, args []string) error { return setDone(args[0], false) },
	}
}

func rmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <id>",
		Short: "Удалить задачу",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			s := mustStore()
			tasks, err := s.Load()
			if err != nil {
				return err
			}
			i, err := store.Find(tasks, args[0])
			if err != nil {
				return err
			}
			id := tasks[i].ID
			tasks = append(tasks[:i], tasks[i+1:]...)
			if err := s.Save(tasks); err != nil {
				return err
			}
			notifyWaybar()
			fmt.Println("удалено:", id)
			return nil
		},
	}
}

func getCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Вывести JSON одной задачи",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			s := mustStore()
			tasks, err := s.Load()
			if err != nil {
				return err
			}
			i, err := store.Find(tasks, args[0])
			if err != nil {
				return err
			}
			return printJSON(tasks[i])
		},
	}
}

func setDone(id string, done bool) error {
	s := mustStore()
	tasks, err := s.Load()
	if err != nil {
		return err
	}
	i, err := store.Find(tasks, id)
	if err != nil {
		return err
	}
	tasks[i].Done = done
	if err := s.Save(tasks); err != nil {
		return err
	}
	notifyWaybar()
	fmt.Println(tasks[i].ID)
	return nil
}

// applyFilter оставляет задачи согласно all/active/done.
func applyFilter(tasks []model.Task, filter string) []model.Task {
	switch strings.ToLower(filter) {
	case "active":
		return filterFunc(tasks, func(t model.Task) bool { return !t.Done })
	case "done":
		return filterFunc(tasks, func(t model.Task) bool { return t.Done })
	default:
		return tasks
	}
}

func filterFunc(tasks []model.Task, keep func(model.Task) bool) []model.Task {
	out := make([]model.Task, 0, len(tasks))
	for _, t := range tasks {
		if keep(t) {
			out = append(out, t)
		}
	}
	return out
}

func parseSort(s string) store.SortMode {
	switch strings.ToLower(s) {
	case "deadline":
		return store.SortDeadline
	case "created":
		return store.SortCreated
	default:
		return store.SortPriority
	}
}

func normalizeTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func formatTaskLine(t model.Task) string {
	box := "[ ]"
	if t.Done {
		box = "[x]"
	}
	parts := []string{t.ID[:8], box, fmt.Sprintf("(%s)", t.Priority), t.Text}
	if len(t.Tags) > 0 {
		parts = append(parts, "["+strings.Join(t.Tags, ",")+"]")
	}
	if label := t.DeadlineLabel(time.Now()); label != "" {
		parts = append(parts, "→ "+label)
	}
	return strings.Join(parts, " ")
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
