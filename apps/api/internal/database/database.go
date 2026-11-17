package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"blood/apps/api/internal/access"
	"blood/apps/api/internal/auth"
	"blood/apps/api/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const CurrentVersion = 2

type SchemaMigration struct {
	Version   uint `gorm:"primaryKey;autoIncrement:false"`
	AppliedAt time.Time
}

func Open(cfg config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}
	pool, err := db.DB()
	if err != nil {
		return nil, err
	}
	pool.SetMaxOpenConns(20)
	pool.SetMaxIdleConns(5)
	pool.SetConnMaxLifetime(30 * time.Minute)
	return db, nil
}

// Migrate применяет версии схемы одной транзакцией только через GORM.
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
		for version := uint(1); version <= CurrentVersion; version++ {
			var applied SchemaMigration
			if err := tx.Where(&SchemaMigration{Version: version}).Take(&applied).Error; err == nil {
				continue
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if version == 2 {
				if err := tx.AutoMigrate(&auth.User{}, &auth.Session{}, &access.Room{}, &access.Membership{}, &access.Conversation{}, &access.ConversationMember{}); err != nil {
					return err
				}
			}
			if err := tx.Create(&SchemaMigration{Version: version, AppliedAt: time.Now().UTC()}).Error; err != nil {
				return err
			}
		}
		return nil
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
