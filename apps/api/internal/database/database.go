package database

import (
	"context"
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
		return db.WithContext(ctx).Where(&SchemaMigration{Version: CurrentVersion}).Take(&migration).Error
	}
}
