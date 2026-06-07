package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Letter struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	User      User       `gorm:"constraint:OnDelete:CASCADE" json:"-"`
	ChildID   *uuid.UUID `gorm:"type:uuid;index" json:"child_id"`
	Child     *Child     `gorm:"constraint:OnDelete:SET NULL" json:"-"`
	Title     string     `json:"title"`
	ImagePath string     `gorm:"not null" json:"-"`
	MimeType  string     `gorm:"not null" json:"mime_type"`
	FileSize  int64      `gorm:"not null" json:"file_size"`
	OCRText   string     `json:"ocr_text"`
	CreatedAt time.Time  `gorm:"not null" json:"created_at"`
}

func (l *Letter) BeforeCreate(_ *gorm.DB) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	return nil
}
