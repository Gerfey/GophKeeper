package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gerfey/gophkeeper/internal/client"
)

func main() {
	configPath := getConfigPath()

	runTUI(configPath)
}

func runTUI(configPath string) {
	if os.Getenv("TERM") == "" {
		if err := os.Setenv("TERM", "xterm-256color"); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка установки переменной окружения TERM: %v\n", err)
		}
	}

	tui, err := client.NewTUI(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка инициализации TUI: %v\n", err)
		os.Exit(1)
	}

	if runErr := tui.Run(); runErr != nil {
		fmt.Fprintf(os.Stderr, "Ошибка запуска TUI: %v\n", runErr)
		os.Exit(1)
	}
}

func getConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка определения домашней директории: %v\n", err)
		os.Exit(1)
	}

	return filepath.Join(homeDir, "config.json")
}
