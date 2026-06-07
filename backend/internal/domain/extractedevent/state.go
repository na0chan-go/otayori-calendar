package extractedevent

import "errors"

const (
	StatusDraft   = "draft"
	StatusIgnored = "ignored"
)

var (
	ErrRegisteredCannotBeConfirmed = errors.New("registered events cannot be confirmed")
	ErrRegisteredCannotBeIgnored   = errors.New("registered events cannot be ignored")
	ErrStatusChanged               = errors.New("event status changed after the original action")
	ErrStatusCannotBeRestored      = errors.New("event status cannot be restored")
)

type State struct {
	Status          string
	CalendarEventID string
}

func (s State) Confirmed() (State, error) {
	if s.Status == StatusRegistered {
		return s, ErrRegisteredCannotBeConfirmed
	}
	s.Status = StatusConfirmed
	s.CalendarEventID = ""
	return s, nil
}

func (s State) Ignored() (State, error) {
	if s.Status == StatusRegistered {
		return s, ErrRegisteredCannotBeIgnored
	}
	s.Status = StatusIgnored
	s.CalendarEventID = ""
	return s, nil
}

func (s State) Restored(expectedStatus, targetStatus string) (State, error) {
	if s.Status != expectedStatus {
		return s, ErrStatusChanged
	}
	if !isLocallyRestorableStatus(s.Status) || !isLocallyRestorableStatus(targetStatus) {
		return s, ErrStatusCannotBeRestored
	}
	s.Status = targetStatus
	return s, nil
}

func isLocallyRestorableStatus(status string) bool {
	return status == StatusDraft || status == StatusConfirmed || status == StatusIgnored
}
