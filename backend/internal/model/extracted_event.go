package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	ExtractedEventStatusDraft      = "draft"
	ExtractedEventStatusConfirmed  = "confirmed"
	ExtractedEventStatusRegistered = "registered"
	ExtractedEventStatusIgnored    = "ignored"
	ExtractedEventStatusFailed     = "failed"
	ExtractedEventStatusDeleted    = "deleted"
)

type ExtractedEvent struct {
	ID                    uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
	LetterID              uuid.UUID       `gorm:"type:uuid;not null;index" json:"letter_id"`
	Letter                Letter          `gorm:"constraint:OnDelete:CASCADE" json:"-"`
	Title                 string          `gorm:"not null" json:"title"`
	EventDate             time.Time       `gorm:"type:date;not null" json:"event_date"`
	StartTime             *datatypes.Time `json:"start_time"`
	EndTime               *datatypes.Time `json:"end_time"`
	IsAllDay              bool            `gorm:"not null;default:true" json:"is_all_day"`
	Location              string          `json:"location"`
	Description           string          `json:"description"`
	Belongings            string          `json:"belongings"`
	SubmissionDeadline    *time.Time      `gorm:"type:date" json:"submission_deadline"`
	Confidence            float64         `gorm:"type:numeric(3,2)" json:"confidence"`
	SourceText            string          `json:"source_text"`
	GoogleCalendarEventID string          `gorm:"index" json:"google_calendar_event_id"`
	Status                string          `gorm:"not null;default:draft;index" json:"status"`
	CreatedAt             time.Time       `gorm:"not null" json:"created_at"`
	UpdatedAt             time.Time       `gorm:"not null" json:"updated_at"`
}

func (e *ExtractedEvent) BeforeCreate(_ *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}
