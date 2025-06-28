package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gerfey/gophkeeper/internal/auth"
	"github.com/gerfey/gophkeeper/internal/server"
	"github.com/gerfey/gophkeeper/pkg/api"
	"github.com/gerfey/gophkeeper/pkg/config"
	"github.com/gerfey/gophkeeper/pkg/logger"
)

var (
	Version = "v1.0.0"
)

func main() {
	log := logger.DefaultLogger()
	log.Info("Запуск сервера GophKeeper версии %s", Version)

	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatal("Ошибка загрузки конфигурации: %v", err)
	}

	repo, err := server.NewPostgresRepository(cfg.Database.GetDSN(), log)
	if err != nil {
		log.Fatal("Ошибка подключения к базе данных: %v", err)
	}
	defer repo.Close()

	if err := repo.InitSchema(); err != nil {
		log.Fatal("Ошибка инициализации схемы базы данных: %v", err)
	}

	encryptionKey := make([]byte, 32)
	if _, err := rand.Read(encryptionKey); err != nil {
		log.Fatal("Ошибка генерации ключа шифрования: %v", err)
	}

	tokenManager := auth.NewJWTManager(cfg.Auth.JWTSecret)

	userService := server.NewUserService(repo, log)
	dataService := server.NewDataService(repo, log, encryptionKey)

	handler := api.NewHandler(tokenManager, userService, dataService, log)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      handler.InitRoutes(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		log.Info("Сервер запущен на %s", srv.Addr)
		var err error

		if cfg.Server.TLSCertFile != "" && cfg.Server.TLSKeyFile != "" {
			log.Info("Используются сертификаты из файлов: %s, %s", cfg.Server.TLSCertFile, cfg.Server.TLSKeyFile)
			err = srv.ListenAndServeTLS(cfg.Server.TLSCertFile, cfg.Server.TLSKeyFile)
		} else {
			log.Warn("TLS не настроен, сервер запущен без шифрования")
			err = srv.ListenAndServe()
		}

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Ошибка запуска сервера: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Завершение работы сервера...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Ошибка при завершении работы сервера: %v", err)
	}

	log.Info("Сервер остановлен")
}
