package extractedevent

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrNotEditable        = errors.New("registered events cannot be edited")
	ErrTitleRequired      = errors.New("title is required")
	ErrInvalidEventDate   = errors.New("event_date must be YYYY-MM-DD")
	ErrInvalidStartTime   = errors.New("start_time must be HH:MM")
	ErrInvalidEndTime     = errors.New("end_time must be HH:MM")
	ErrTimedRangeRequired = errors.New("start_time and end_time are required when is_all_day is false")
	ErrInvalidTimedRange  = errors.New("end_time must be after start_time")
	ErrInvalidDeadline    = errors.New("submission_deadline must be YYYY-MM-DD")
)

type Candidate struct {
	Title              string
	EventDate          string
	StartTime          *string
	EndTime            *string
	IsAllDay           bool
	Location           string
	Description        string
	Belongings         string
	SubmissionDeadline *string
	Status             string
}

type Update struct {
	Title              *string
	EventDate          *string
	StartTime          *string
	EndTime            *string
	IsAllDay           *bool
	Location           *string
	Description        *string
	Belongings         *string
	SubmissionDeadline *string
}

func (c Candidate) Updated(update Update) (Candidate, error) {
	if err := c.ValidateEditable(); err != nil {
		return c, err
	}

	if update.Title != nil {
		c.Title = strings.TrimSpace(*update.Title)
		if c.Title == "" {
			return c, ErrTitleRequired
		}
	}
	if update.EventDate != nil {
		c.EventDate = strings.TrimSpace(*update.EventDate)
		if !validDate(c.EventDate) {
			return c, ErrInvalidEventDate
		}
	}
	if update.StartTime != nil {
		value, err := optionalClock(update.StartTime, ErrInvalidStartTime)
		if err != nil {
			return c, err
		}
		c.StartTime = value
	}
	if update.EndTime != nil {
		value, err := optionalClock(update.EndTime, ErrInvalidEndTime)
		if err != nil {
			return c, err
		}
		c.EndTime = value
	}
	if update.IsAllDay != nil {
		c.IsAllDay = *update.IsAllDay
	}

	if c.IsAllDay {
		c.StartTime = nil
		c.EndTime = nil
	} else {
		if c.StartTime == nil || c.EndTime == nil {
			return c, ErrTimedRangeRequired
		}
		if *c.EndTime <= *c.StartTime {
			return c, ErrInvalidTimedRange
		}
	}

	if update.Location != nil {
		c.Location = strings.TrimSpace(*update.Location)
	}
	if update.Description != nil {
		c.Description = strings.TrimSpace(*update.Description)
	}
	if update.Belongings != nil {
		c.Belongings = strings.TrimSpace(*update.Belongings)
	}
	if update.SubmissionDeadline != nil {
		value := strings.TrimSpace(*update.SubmissionDeadline)
		if value == "" {
			c.SubmissionDeadline = nil
		} else {
			if !validDate(value) {
				return c, ErrInvalidDeadline
			}
			c.SubmissionDeadline = &value
		}
	}

	if c.Status != StatusDeleted {
		c.Status = StatusConfirmed
	}
	return c, nil
}

func (c Candidate) ValidateEditable() error {
	if c.Status == StatusRegistered {
		return ErrNotEditable
	}
	return nil
}

func optionalClock(value *string, invalidErr error) (*string, error) {
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil, nil
	}
	if _, err := time.Parse("15:04", trimmed); err != nil {
		return nil, invalidErr
	}
	return &trimmed, nil
}

func validDate(value string) bool {
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}
