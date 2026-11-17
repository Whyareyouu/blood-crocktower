package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"blood/apps/api/internal/config"
	"blood/apps/api/internal/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Некорректная конфигурация", "error", err)
		os.Exit(1)
	}
	db, err := database.Open(cfg)
	if err != nil {
		slog.Error("Не удалось подключиться к PostgreSQL; проверьте настройки DB_* и доступность БД")
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := database.Migrate(ctx, db); err != nil {
		slog.Error("Не удалось применить миграцию", "error", err)
		os.Exit(1)
	}
	slog.Info("Миграции применены", "version", database.CurrentVersion)
}
