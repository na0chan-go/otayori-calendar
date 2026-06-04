package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	GoogleUserID string    `gorm:"uniqueIndex;not null" json:"-"`
	Email        string    `gorm:"not null" json:"email"`
	Name         string    `json:"name"`
	CreatedAt    time.Time `gorm:"not null" json:"created_at"`
}

func (u *User) BeforeCreate(_ *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
