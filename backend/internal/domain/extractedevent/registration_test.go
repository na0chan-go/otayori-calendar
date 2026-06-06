package extractedevent

import (
	"errors"
	"testing"
)

func TestRegistrationValidate(t *testing.T) {
	for _, status := range []string{StatusConfirmed, StatusFailed, StatusDeleted} {
		t.Run(status, func(t *testing.T) {
			if err := (Registration{Status: status}).Validate(); err != nil {
				t.Fatalf("expected %s to be registerable, got %v", status, err)
			}
		})
	}

	if err := (Registration{Status: "draft"}).Validate(); !errors.Is(err, ErrNotRegisterable) {
		t.Fatalf("expected not registerable error, got %v", err)
	}
	if err := (Registration{Status: StatusRegistered, CalendarEventID: "event-id"}).Validate(); !errors.Is(err, ErrAlreadyRegistered) {
		t.Fatalf("expected already registered error, got %v", err)
	}
}

func TestRegistrationTransitions(t *testing.T) {
	registration := Registration{Status: StatusConfirmed}

	failed := registration.Failed()
	if failed.Status != StatusFailed {
		t.Fatalf("expected failed status, got %q", failed.Status)
	}

	registered, err := registration.Registered("event-id")
	if err != nil {
		t.Fatal(err)
	}
	if registered.Status != StatusRegistered || registered.CalendarEventID != "event-id" {
		t.Fatalf("unexpected registered state: %#v", registered)
	}

	missingID, err := registration.Registered("")
	if !errors.Is(err, ErrCalendarEventIDMissing) || missingID.Status != StatusFailed {
		t.Fatalf("expected failed state and missing id error, got %#v, %v", missingID, err)
	}
}
