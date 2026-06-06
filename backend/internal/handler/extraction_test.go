package handler

import (
	"testing"

	"github.com/google/uuid"
)

func TestValidateExtractionOutput(t *testing.T) {
	start := "15:00"
	end := "16:00"
	deadline := "2026-06-10"
	events, err := validateExtractionOutput(uuid.New(), extractionOutput{
		Events: []extractionEvent{
			{
				Title:      "保護者会",
				Date:       "2026-06-12",
				StartTime:  &start,
				EndTime:    &end,
				Belongings: "スリッパ",
				Deadline:   &deadline,
				Confidence: 0.88,
				SourceText: "6月12日（金）保護者会があります。",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(events) != 1 {
		t.Fatalf("expected one event, got %d", len(events))
	}
	if events[0].Status != "draft" {
		t.Fatalf("expected draft status, got %q", events[0].Status)
	}
	if events[0].StartTime == nil || events[0].StartTime.String() != "15:00:00" {
		t.Fatalf("expected normalized start time, got %#v", events[0].StartTime)
	}
	if events[0].Belongings != "スリッパ" {
		t.Fatalf("expected belongings, got %q", events[0].Belongings)
	}
	if events[0].SubmissionDeadline == nil || events[0].SubmissionDeadline.Format("2006-01-02") != deadline {
		t.Fatalf("expected submission deadline, got %#v", events[0].SubmissionDeadline)
	}
}

func TestValidateExtractionOutputRejectsInvalidConfidence(t *testing.T) {
	_, err := validateExtractionOutput(uuid.New(), extractionOutput{
		Events: []extractionEvent{
			{
				Title:      "身体測定",
				Date:       "2026-06-12",
				Confidence: 1.2,
			},
		},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateExtractionOutputRejectsInvalidSubmissionDeadline(t *testing.T) {
	deadline := "6月10日"
	_, err := validateExtractionOutput(uuid.New(), extractionOutput{
		Events: []extractionEvent{
			{
				Title:      "保護者会",
				Date:       "2026-06-12",
				Deadline:   &deadline,
				Confidence: 0.8,
			},
		},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestExtractEventsFromOCRText(t *testing.T) {
	output := extractEventsFromOCRText("6月12日（金）身体測定を行います。\n朝は薄着で登園してください。", 2026)

	if len(output.Events) != 1 {
		t.Fatalf("expected one event, got %d", len(output.Events))
	}
	event := output.Events[0]
	if event.Date != "2026-06-12" {
		t.Fatalf("expected date 2026-06-12, got %q", event.Date)
	}
	if event.Title != "身体測定" {
		t.Fatalf("expected title, got %q", event.Title)
	}
	if !event.IsAllDay {
		t.Fatal("expected all-day event")
	}
	if event.SourceText == "" {
		t.Fatal("expected source text")
	}
}
