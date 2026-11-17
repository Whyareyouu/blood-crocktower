// Package testutil предоставляет изолированную настоящую PostgreSQL для тестов.
package testutil

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"blood/apps/api/internal/access"
	"blood/apps/api/internal/auth"
	"blood/apps/api/internal/config"
	"blood/apps/api/internal/database"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func Database(t *testing.T) *gorm.DB {
	t.Helper()
	if os.Getenv("TEST_DATABASE") != "1" {
		t.Skip("для интеграционной проверки PostgreSQL задайте TEST_DATABASE=1")
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL()), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		NamingStrategy: schema.NamingStrategy{TablePrefix: fmt.Sprintf("s2_%d_", time.Now().UnixNano())},
	})
	if err != nil {
		t.Fatal("не удалось открыть тестовую БД")
	}
	t.Cleanup(func() {
		if err := db.Migrator().DropTable(&access.ConversationMember{}, &access.Conversation{}, &access.Membership{}, &access.Room{}, &auth.Session{}, &auth.User{}, &database.SchemaMigration{}); err != nil {
			t.Error(err)
		}
		pool, err := db.DB()
		if err == nil {
			_ = pool.Close()
		}
	})
	if err := database.Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	return db
}
