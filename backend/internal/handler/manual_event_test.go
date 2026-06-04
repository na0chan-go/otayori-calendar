package handler

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBuildManualEventAllDay(t *testing.T) {
	server := newTestServer()
	userID := uuid.New()
	isAllDay := true

	manualEvent, googleEvent, err := server.buildManualEvent(userID, manualEventRequest{
		Title:       "身体測定",
		EventDate:   "2026-06-12",
		IsAllDay:    &isAllDay,
		Location:    "保育園",
		Description: "持ち物：体操服",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !manualEvent.IsAllDay {
		t.Fatal("expected manual event to be all-day")
	}
	if manualEvent.TimeZone != "Asia/Tokyo" {
		t.Fatalf("expected default timezone, got %q", manualEvent.TimeZone)
	}
	if googleEvent.Start.Date != "2026-06-12" {
		t.Fatalf("expected start date, got %q", googleEvent.Start.Date)
	}
	if googleEvent.End.Date != "2026-06-13" {
		t.Fatalf("expected end date to be next day, got %q", googleEvent.End.Date)
	}
	if googleEvent.Start.DateTime != "" || googleEvent.End.DateTime != "" {
		t.Fatal("all-day event should not set dateTime")
	}
}

func TestBuildManualEventTimed(t *testing.T) {
	server := newTestServer()
	userID := uuid.New()
	isAllDay := false

	manualEvent, googleEvent, err := server.buildManualEvent(userID, manualEventRequest{
		Title:     "保護者会",
		EventDate: "2026-06-12",
		StartTime: "15:00",
		EndTime:   "16:00",
		IsAllDay:  &isAllDay,
		TimeZone:  "Asia/Tokyo",
	})
	if err != nil {
		t.Fatal(err)
	}

	if manualEvent.IsAllDay {
		t.Fatal("expected timed manual event")
	}
	if manualEvent.StartAt == nil || manualEvent.EndAt == nil {
		t.Fatal("expected start and end times to be stored")
	}

	wantStart := time.Date(2026, 6, 12, 15, 0, 0, 0, time.FixedZone("JST", 9*60*60)).Format(time.RFC3339)
	wantEnd := time.Date(2026, 6, 12, 16, 0, 0, 0, time.FixedZone("JST", 9*60*60)).Format(time.RFC3339)
	if googleEvent.Start.DateTime != wantStart {
		t.Fatalf("expected start dateTime %q, got %q", wantStart, googleEvent.Start.DateTime)
	}
	if googleEvent.End.DateTime != wantEnd {
		t.Fatalf("expected end dateTime %q, got %q", wantEnd, googleEvent.End.DateTime)
	}
	if googleEvent.Start.Date != "" || googleEvent.End.Date != "" {
		t.Fatal("timed event should not set date")
	}
}

func TestBuildManualEventRejectsInvalidTimeRange(t *testing.T) {
	server := newTestServer()
	isAllDay := false

	_, _, err := server.buildManualEvent(uuid.New(), manualEventRequest{
		Title:     "保護者会",
		EventDate: "2026-06-12",
		StartTime: "16:00",
		EndTime:   "15:00",
		IsAllDay:  &isAllDay,
	})
	if err == nil {
		t.Fatal("expected invalid time range error")
	}
}
