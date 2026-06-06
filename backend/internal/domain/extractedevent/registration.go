package extractedevent

import "errors"

const (
	StatusConfirmed  = "confirmed"
	StatusRegistered = "registered"
	StatusFailed     = "failed"
	StatusDeleted    = "deleted"
)

var (
	ErrAlreadyRegistered      = errors.New("event is already registered")
	ErrNotRegisterable        = errors.New("only confirmed, failed, or deleted events can be registered")
	ErrCalendarEventIDMissing = errors.New("google calendar event id is missing")
)

type Registration struct {
	Status          string
	CalendarEventID string
}

func (r Registration) Validate() error {
	if r.CalendarEventID != "" && r.Status != StatusDeleted {
		return ErrAlreadyRegistered
	}
	if r.Status != StatusConfirmed && r.Status != StatusFailed && r.Status != StatusDeleted {
		return ErrNotRegisterable
	}
	return nil
}

func (r Registration) Failed() Registration {
	r.Status = StatusFailed
	return r
}

func (r Registration) Registered(calendarEventID string) (Registration, error) {
	if calendarEventID == "" {
		return r.Failed(), ErrCalendarEventIDMissing
	}
	r.Status = StatusRegistered
	r.CalendarEventID = calendarEventID
	return r, nil
}
