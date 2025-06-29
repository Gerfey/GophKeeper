package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gerfey/gophkeeper/internal/client"
)

func main() {
	configPath, err := getConfigPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка определения пути конфигурации: %v\n", err)
		os.Exit(1)
	}

	if errTUI := runTUI(configPath); errTUI != nil {
		fmt.Fprintf(os.Stderr, "Ошибка при работе приложения: %v\n", errTUI)
		os.Exit(1)
	}
}

func runTUI(configPath string) error {
	if os.Getenv("TERM") == "" {
		if err := os.Setenv("TERM", "xterm-256color"); err != nil {
			return fmt.Errorf("ошибка установки переменной окружения TERM: %w", err)
		}
	}

	tui, err := client.NewTUI(configPath)
	if err != nil {
		return fmt.Errorf("ошибка инициализации TUI: %w", err)
	}

	if runErr := tui.Run(); runErr != nil {
		return fmt.Errorf("ошибка запуска TUI: %w", runErr)
	}

	return nil
}

func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("ошибка определения домашней директории: %w", err)
	}

	return filepath.Join(homeDir, "config.json"), nil
}
