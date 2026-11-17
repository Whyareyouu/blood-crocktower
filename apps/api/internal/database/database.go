package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"blood/apps/api/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const CurrentVersion = 1

type SchemaMigration struct {
	Version   uint `gorm:"primaryKey;autoIncrement:false"`
	AppliedAt time.Time
}

func Open(cfg config.Config) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(cfg.DatabaseURL()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
}

// Migrate создаёт начальный журнал схемы. Доменные таблицы добавляются
// отдельными версионированными миграциями при реализации соответствующих этапов.
func Migrate(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if tx.Migrator().HasTable(&SchemaMigration{}) {
			var latest SchemaMigration
			if err := tx.Order("version DESC").Take(&latest).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if latest.Version > CurrentVersion {
				return fmt.Errorf("версия схемы %d новее поддерживаемой %d", latest.Version, CurrentVersion)
			}
		}
		if err := tx.AutoMigrate(&SchemaMigration{}); err != nil {
			return err
		}
		return tx.Where(&SchemaMigration{Version: CurrentVersion}).FirstOrCreate(
			&SchemaMigration{Version: CurrentVersion, AppliedAt: time.Now().UTC()},
		).Error
	})
}

// Ready проверяет БД и применённую версию схемы только через ORM.
func Ready(db *gorm.DB) func(context.Context) error {
	return func(ctx context.Context) error {
		var migration SchemaMigration
		if err := db.WithContext(ctx).Order("version DESC").Take(&migration).Error; err != nil {
			return err
		}
		if migration.Version != CurrentVersion {
			return fmt.Errorf("версия схемы %d не соответствует поддерживаемой %d", migration.Version, CurrentVersion)
		}
		return nil
	}
}
