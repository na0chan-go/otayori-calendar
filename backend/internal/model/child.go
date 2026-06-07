package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Child struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"-"`
	Name      string    `gorm:"not null" json:"name"`
	Color     string    `gorm:"not null;default:#8fcfb0" json:"color"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null" json:"updated_at"`
}

func (c *Child) BeforeCreate(_ *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}
