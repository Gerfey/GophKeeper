package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gerfey/gophkeeper/internal/client"
	"github.com/gerfey/gophkeeper/pkg/config"
	"github.com/gerfey/gophkeeper/pkg/logger"
)

func main() {
	log := logger.DefaultLogger()
	log.Infof("Запуск клиента GophKeeper")

	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Errorf("Ошибка загрузки конфигурации: %v", err)
		os.Exit(1)
	}

	if errTUI := runTUI(cfg); errTUI != nil {
		log.Errorf("Ошибка при работе приложения: %v", errTUI)
		os.Exit(1)
	}
}

func runTUI(cfg *config.Config) error {
	if os.Getenv("TERM") == "" {
		if err := os.Setenv("TERM", "xterm-256color"); err != nil {
			return fmt.Errorf("ошибка установки переменной окружения TERM: %w", err)
		}
	}

	tui, err := client.NewTUI(cfg)
	if err != nil {
		return fmt.Errorf("ошибка инициализации TUI: %w", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		tui.Stop()
		os.Exit(0)
	}()

	if runErr := tui.Run(); runErr != nil {
		return fmt.Errorf("ошибка запуска TUI: %w", runErr)
	}

	return nil
}
