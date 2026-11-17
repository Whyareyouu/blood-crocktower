package database

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"blood/apps/api/internal/access"
	"blood/apps/api/internal/auth"
	"blood/apps/api/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// Каждый тест работает только со своими таблицами, в том числе в CI.
// Существующая схема приложения и её данные не изменяются.
func isolatedDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	if os.Getenv("TEST_DATABASE") != "1" {
		t.Skip("для интеграционной проверки PostgreSQL задайте TEST_DATABASE=1")
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: fmt.Sprintf("stage1_test_%d_", time.Now().UnixNano()),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := db.WithContext(ctx).Migrator().DropTable(&access.ConversationMember{}, &access.Conversation{}, &access.Membership{}, &access.Room{}, &auth.Session{}, &auth.User{}, &SchemaMigration{}); err != nil {
			t.Errorf("не удалось удалить временную таблицу: %v", err)
		}
		pool, err := db.DB()
		if err == nil {
			_ = pool.Close()
		}
	})
	return db
}

func TestMigrationAndReadiness(t *testing.T) {
	db := isolatedDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := Ready(db)(ctx); err == nil {
		t.Fatal("БД без миграции не должна быть готова")
	}
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	var first SchemaMigration
	if err := db.WithContext(ctx).Order("version").Take(&first).Error; err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := Ready(db)(ctx); err != nil {
		t.Fatal(err)
	}
	var migrations []SchemaMigration
	if err := db.WithContext(ctx).Order("version").Find(&migrations).Error; err != nil {
		t.Fatal(err)
	}
	if len(migrations) != CurrentVersion || migrations[len(migrations)-1].Version != CurrentVersion || !migrations[0].AppliedAt.Equal(first.AppliedAt) {
		t.Fatalf("повторная миграция изменила журнал: %+v", migrations)
	}
}

func TestEmptyJournalIsNotReady(t *testing.T) {
	db := isolatedDatabase(t)
	if err := db.AutoMigrate(&SchemaMigration{}); err != nil {
		t.Fatal(err)
	}
	if err := Ready(db)(context.Background()); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("пустой журнал не должен означать готовность: %v", err)
	}
}

func TestNewerSchemaIsRejected(t *testing.T) {
	db := isolatedDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := db.WithContext(ctx).Create(&SchemaMigration{Version: CurrentVersion + 1, AppliedAt: time.Now().UTC()}).Error; err != nil {
		t.Fatal(err)
	}
	if err := Ready(db)(ctx); err == nil {
		t.Fatal("старый сервер не должен работать с более новой схемой")
	}
	if err := Migrate(ctx, db); err == nil {
		t.Fatal("старый migrator не должен принимать более новую схему")
	}
	var count int64
	if err := db.WithContext(ctx).Model(&SchemaMigration{}).Count(&count).Error; err != nil || count != CurrentVersion+1 {
		t.Fatalf("журнал новой версии изменён: count=%d, err=%v", count, err)
	}
}

func TestCanceledMigrationDoesNotCreateSchema(t *testing.T) {
	db := isolatedDatabase(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Migrate(ctx, db); !errors.Is(err, context.Canceled) {
		t.Fatalf("миграция должна учитывать отмену: %v", err)
	}
	if db.Migrator().HasTable(&SchemaMigration{}) {
		t.Fatal("отменённая миграция создала таблицу")
	}
	if err := Migrate(context.Background(), db); err != nil {
		t.Fatalf("повторный запуск после отмены должен работать: %v", err)
	}
}

func TestUpgradeFromStageOnePreservesJournal(t *testing.T) {
	db := isolatedDatabase(t)
	if err := db.AutoMigrate(&SchemaMigration{}); err != nil {
		t.Fatal(err)
	}
	first := SchemaMigration{Version: 1, AppliedAt: time.Now().UTC().Truncate(time.Microsecond)}
	if err := db.Create(&first).Error; err != nil {
		t.Fatal(err)
	}
	if err := Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	var saved SchemaMigration
	if err := db.Where(&SchemaMigration{Version: 1}).Take(&saved).Error; err != nil {
		t.Fatal(err)
	}
	if !saved.AppliedAt.Equal(first.AppliedAt) || !db.Migrator().HasTable(&auth.Session{}) || !db.Migrator().HasTable(&access.Membership{}) {
		t.Fatal("обновление потеряло журнал или не создало таблицы")
	}
	if err := Ready(db)(context.Background()); err != nil {
		t.Fatal(err)
	}
}
