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

	_ "github.com/lib/pq"
)

const (
	version = "v1.0.0"

	shutdownTimeoutSec = 5
)

func main() {
	log := logger.DefaultLogger()
	log.Infof("Запуск сервера GophKeeper версии %s", version)

	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	repo, err := server.NewPostgresRepository(cfg.Database.GetDSN(), log)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer repo.Close()

	if initErr := repo.InitSchema(); initErr != nil {
		log.Errorf("Ошибка инициализации схемы базы данных: %v", initErr)

		return
	}

	encryptionKey := make([]byte, cfg.Encryption.KeySize)
	if _, randErr := rand.Read(encryptionKey); randErr != nil {
		log.Errorf("Ошибка генерации ключа шифрования: %v", randErr)

		return
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
		log.Infof("Сервер запущен на %s", srv.Addr)
		var serverErr error

		if cfg.Server.TLSCertFile != "" && cfg.Server.TLSKeyFile != "" {
			log.Infof("Используются сертификаты из файлов: %s, %s", cfg.Server.TLSCertFile, cfg.Server.TLSKeyFile)
			serverErr = srv.ListenAndServeTLS(cfg.Server.TLSCertFile, cfg.Server.TLSKeyFile)
		} else {
			log.Warnf("TLS не настроен, сервер запущен без шифрования")
			serverErr = srv.ListenAndServe()
		}

		if serverErr != nil && !errors.Is(serverErr, http.ErrServerClosed) {
			log.Fatalf("Ошибка запуска сервера: %v", serverErr)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Infof("Завершение работы сервера...")
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeoutSec*time.Second)
	defer cancel()

	if shutdownErr := srv.Shutdown(ctx); shutdownErr != nil {
		log.Errorf("Ошибка при завершении работы сервера: %v", shutdownErr)
	}

	log.Infof("Сервер остановлен")
}
