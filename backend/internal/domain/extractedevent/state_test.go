package extractedevent

import (
	"errors"
	"testing"
)

func TestStateConfirmAndIgnore(t *testing.T) {
	state := State{Status: StatusDraft, CalendarEventID: "stale-id"}

	confirmed, err := state.Confirmed()
	if err != nil || confirmed.Status != StatusConfirmed || confirmed.CalendarEventID != "" {
		t.Fatalf("unexpected confirmed state: %#v, %v", confirmed, err)
	}
	ignored, err := state.Ignored()
	if err != nil || ignored.Status != StatusIgnored || ignored.CalendarEventID != "" {
		t.Fatalf("unexpected ignored state: %#v, %v", ignored, err)
	}

	registered := State{Status: StatusRegistered}
	if _, err := registered.Confirmed(); !errors.Is(err, ErrRegisteredCannotBeConfirmed) {
		t.Fatalf("expected confirm error, got %v", err)
	}
	if _, err := registered.Ignored(); !errors.Is(err, ErrRegisteredCannotBeIgnored) {
		t.Fatalf("expected ignore error, got %v", err)
	}
}

func TestStateRestore(t *testing.T) {
	state := State{Status: StatusIgnored}
	restored, err := state.Restored(StatusIgnored, StatusDraft)
	if err != nil || restored.Status != StatusDraft {
		t.Fatalf("unexpected restored state: %#v, %v", restored, err)
	}

	if _, err := state.Restored(StatusConfirmed, StatusDraft); !errors.Is(err, ErrStatusChanged) {
		t.Fatalf("expected changed error, got %v", err)
	}
	if _, err := state.Restored(StatusIgnored, StatusDeleted); !errors.Is(err, ErrStatusCannotBeRestored) {
		t.Fatalf("expected restore error, got %v", err)
	}
}
