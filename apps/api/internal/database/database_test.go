package database

import (
	"context"
	"os"
	"testing"
	"time"

	"blood/apps/api/internal/config"
)

func TestMigrationAndReadiness(t *testing.T) {
	if os.Getenv("TEST_DATABASE") != "1" {
		t.Skip("для интеграционной проверки PostgreSQL задайте TEST_DATABASE=1")
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	db, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	for range 2 {
		if err := Migrate(ctx, db); err != nil {
			t.Fatal(err)
		}
	}
	if err := Ready(db)(ctx); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.WithContext(ctx).Model(&SchemaMigration{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("миграция должна быть идемпотентной: count=%d, err=%v", count, err)
	}
}
