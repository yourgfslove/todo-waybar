package cmd

import (
	"fmt"
	"os"

	"gtodo/internal/store"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gtodo",
	Short: "Todo для Arch/Hyprland: TUI, Waybar и CLI в одном бинаре",
	Long: "gtodo — единый бинарь с тремя режимами:\n" +
		"  gtodo tui      интерактивный TUI (Bubbletea)\n" +
		"  gtodo waybar   JSON для модуля Waybar\n" +
		"  gtodo add/list/done/undone/rm/get   управление из CLI",
}

// Execute запускает корневую команду.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "ошибка:", err)
		os.Exit(1)
	}
}

// mustStore открывает хранилище или завершает программу с ошибкой.
func mustStore() *store.Store {
	s, err := store.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ошибка хранилища:", err)
		os.Exit(1)
	}
	return s
}
