package database

import (
	"github.com/na0chan-go/otayori-calendar/backend/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Open(databaseURL string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
		&model.GoogleToken{},
		&model.Child{},
		&model.Letter{},
		&model.ManualEvent{},
		&model.ExtractedEvent{},
	)
}
