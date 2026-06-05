package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	ManualEventStatusRegistered = "registered"
	ManualEventStatusFailed     = "failed"
	ManualEventStatusDeleted    = "deleted"
)

type ManualEvent struct {
	ID                    uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID                uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	User                  User       `gorm:"constraint:OnDelete:CASCADE" json:"-"`
	Title                 string     `gorm:"not null" json:"title"`
	EventDate             time.Time  `gorm:"type:date;not null" json:"event_date"`
	StartAt               *time.Time `json:"start_at"`
	EndAt                 *time.Time `json:"end_at"`
	IsAllDay              bool       `gorm:"not null;default:true" json:"is_all_day"`
	Location              string     `json:"location"`
	Description           string     `json:"description"`
	TimeZone              string     `gorm:"not null;default:Asia/Tokyo" json:"time_zone"`
	GoogleCalendarEventID string     `gorm:"index" json:"google_calendar_event_id"`
	Status                string     `gorm:"not null;default:registered;index" json:"status"`
	CreatedAt             time.Time  `gorm:"not null" json:"created_at"`
	UpdatedAt             time.Time  `gorm:"not null" json:"updated_at"`
}

func (e *ManualEvent) BeforeCreate(_ *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}
