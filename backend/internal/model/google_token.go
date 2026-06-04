package model

import (
	"time"

	"github.com/google/uuid"
)

type GoogleToken struct {
	UserID       uuid.UUID `gorm:"type:uuid;primaryKey"`
	User         User      `gorm:"constraint:OnDelete:CASCADE"`
	AccessToken  string    `gorm:"not null"`
	RefreshToken string
	Expiry       time.Time `gorm:"not null"`
	CreatedAt    time.Time `gorm:"not null"`
	UpdatedAt    time.Time `gorm:"not null"`
}
