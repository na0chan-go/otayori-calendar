package handler

import (
	"testing"

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
