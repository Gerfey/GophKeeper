package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gerfey/gophkeeper/internal/client"
)

var (
	version = "v1.0.0"
)

func main() {
	client.Version = version

	configPath := getConfigPath()

	runTUI(configPath)
}

func runTUI(configPath string) {
	if os.Getenv("TERM") == "" {
		os.Setenv("TERM", "xterm-256color")
	}

	tui, err := client.NewTUI(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка инициализации TUI: %v\n", err)
		os.Exit(1)
	}

	if err := tui.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка запуска TUI: %v\n", err)
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
