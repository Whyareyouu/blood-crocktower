package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"blood/apps/api/internal/access"
	"blood/apps/api/internal/auth"
	"blood/apps/api/internal/config"
	"blood/apps/api/internal/database"
	"blood/apps/api/internal/httpapi"
)

func main() {
	if err := run(); err != nil {
		slog.Error("Сервер остановлен", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	db, err := database.Open(cfg)
	if err != nil {
		return errors.New("не удалось подключиться к PostgreSQL; проверьте настройки DB_* и доступность БД")
	}
	authService, err := auth.New(db)
	if err != nil {
		return errors.New("не удалось инициализировать авторизацию")
	}
	handler := httpapi.New(database.Ready(db), httpapi.Options{Auth: authService, Access: access.New(db), PublicOrigin: cfg.PublicOrigin, CookieSecure: cfg.CookieSecure})
	defer handler.Close()
	server := &http.Server{
		Addr: fmt.Sprintf(":%d", cfg.APIPort), Handler: handler,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("API запущен", "port", cfg.APIPort)
		serverErrors <- server.ListenAndServe()
	}()
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
	return nil
}
