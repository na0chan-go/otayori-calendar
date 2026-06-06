package handler

import (
	"errors"
	"testing"
	"time"

	"github.com/na0chan-go/otayori-calendar/backend/internal/model"
	"gorm.io/datatypes"
)

func TestApplyExtractedEventUpdateConfirmsAndClearsAllDayTimes(t *testing.T) {
	start := datatypes.NewTime(9, 0, 0, 0)
	end := datatypes.NewTime(10, 0, 0, 0)
	isAllDay := true
	title := " 身体測定 "

	event := model.ExtractedEvent{
		Title:     "仮タイトル",
		StartTime: &start,
		EndTime:   &end,
		Status:    model.ExtractedEventStatusDraft,
	}

	err := applyExtractedEventUpdate(&event, updateExtractedEventRequest{
		Title:    &title,
		IsAllDay: &isAllDay,
	})
	if err != nil {
		t.Fatal(err)
	}

	if event.Title != "身体測定" {
		t.Fatalf("expected trimmed title, got %q", event.Title)
	}
	if event.StartTime != nil || event.EndTime != nil {
		t.Fatal("all-day event should clear times")
	}
	if event.Status != model.ExtractedEventStatusConfirmed {
		t.Fatalf("expected confirmed status, got %q", event.Status)
	}
}

func TestApplyExtractedEventUpdateRejectsInvalidTimedRange(t *testing.T) {
	isAllDay := false
	startTime := "16:00"
	endTime := "15:00"

	event := model.ExtractedEvent{Status: model.ExtractedEventStatusDraft}
	err := applyExtractedEventUpdate(&event, updateExtractedEventRequest{
		StartTime: &startTime,
		EndTime:   &endTime,
		IsAllDay:  &isAllDay,
	})
	if err == nil {
		t.Fatal("expected invalid time range error")
	}
}

func TestValidateExtractedEventEditableRejectsRegisteredEvent(t *testing.T) {
	err := validateExtractedEventEditable(model.ExtractedEvent{Status: model.ExtractedEventStatusRegistered})
	if !errors.Is(err, errExtractedEventNotEditable) {
		t.Fatalf("expected not editable error, got %v", err)
	}
}

func TestValidateExtractedEventEditableAllowsUnregisteredEvent(t *testing.T) {
	err := validateExtractedEventEditable(model.ExtractedEvent{Status: model.ExtractedEventStatusConfirmed})
	if err != nil {
		t.Fatalf("expected confirmed event to be editable, got %v", err)
	}
}

func TestValidateExtractedEventStatusRestoreAllowsLocalStatuses(t *testing.T) {
	err := validateExtractedEventStatusRestore(
		model.ExtractedEventStatusIgnored,
		model.ExtractedEventStatusIgnored,
		model.ExtractedEventStatusDraft,
	)
	if err != nil {
		t.Fatalf("expected local status restore to be allowed, got %v", err)
	}
}

func TestValidateExtractedEventStatusRestoreRejectsChangedOrCalendarStatuses(t *testing.T) {
	if err := validateExtractedEventStatusRestore(
		model.ExtractedEventStatusConfirmed,
		model.ExtractedEventStatusIgnored,
		model.ExtractedEventStatusDraft,
	); err == nil {
		t.Fatal("expected changed current status to be rejected")
	}

	if err := validateExtractedEventStatusRestore(
		model.ExtractedEventStatusConfirmed,
		model.ExtractedEventStatusConfirmed,
		model.ExtractedEventStatusDeleted,
	); err == nil {
		t.Fatal("expected calendar-linked target status to be rejected")
	}
}

func TestBuildGoogleEventFromExtractedEventAllDay(t *testing.T) {
	server := newTestServer()

	googleEvent, err := server.buildGoogleEventFromExtractedEvent(model.ExtractedEvent{
		Title:     "身体測定",
		EventDate: time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC),
		IsAllDay:  true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if googleEvent.Start.Date != "2026-06-12" {
		t.Fatalf("expected start date, got %q", googleEvent.Start.Date)
	}
	if googleEvent.End.Date != "2026-06-13" {
		t.Fatalf("expected next-day end date, got %q", googleEvent.End.Date)
	}
}

func TestBuildGoogleEventFromExtractedEventTimed(t *testing.T) {
	server := newTestServer()
	startTime := datatypes.NewTime(15, 0, 0, 0)
	endTime := datatypes.NewTime(16, 0, 0, 0)

	googleEvent, err := server.buildGoogleEventFromExtractedEvent(model.ExtractedEvent{
		Title:     "保護者会",
		EventDate: time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC),
		StartTime: &startTime,
		EndTime:   &endTime,
		IsAllDay:  false,
	})
	if err != nil {
		t.Fatal(err)
	}

	if googleEvent.Start.DateTime != "2026-06-12T15:00:00+09:00" {
		t.Fatalf("expected start dateTime, got %q", googleEvent.Start.DateTime)
	}
	if googleEvent.End.DateTime != "2026-06-12T16:00:00+09:00" {
		t.Fatalf("expected end dateTime, got %q", googleEvent.End.DateTime)
	}
}

func TestBuildGoogleEventFromExtractedEventRejectsIncompleteTimedEvent(t *testing.T) {
	server := newTestServer()

	_, err := server.buildGoogleEventFromExtractedEvent(model.ExtractedEvent{
		Title:     "保護者会",
		EventDate: time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC),
		IsAllDay:  false,
	})
	if err == nil {
		t.Fatal("expected incomplete timed event error")
	}
}

func TestValidateExtractedEventRegisterableAllowsConfirmedFailedAndDeleted(t *testing.T) {
	for _, status := range []string{
		model.ExtractedEventStatusConfirmed,
		model.ExtractedEventStatusFailed,
		model.ExtractedEventStatusDeleted,
	} {
		t.Run(status, func(t *testing.T) {
			err := validateExtractedEventRegisterable(model.ExtractedEvent{Status: status})
			if err != nil {
				t.Fatalf("expected %s to be registerable, got %v", status, err)
			}
		})
	}
}

func TestValidateExtractedEventRegisterableRejectsDraftAndRegistered(t *testing.T) {
	err := validateExtractedEventRegisterable(model.ExtractedEvent{Status: model.ExtractedEventStatusDraft})
	if !errors.Is(err, errExtractedEventNotRegisterable) {
		t.Fatalf("expected not registerable error, got %v", err)
	}

	err = validateExtractedEventRegisterable(model.ExtractedEvent{
		Status:                model.ExtractedEventStatusRegistered,
		GoogleCalendarEventID: "google-event-id",
	})
	if !errors.Is(err, errExtractedEventAlreadyRegistered) {
		t.Fatalf("expected already registered error, got %v", err)
	}
}
