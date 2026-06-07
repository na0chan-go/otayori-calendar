package extractedevent

import (
	"errors"
	"testing"
)

func TestCandidateUpdated(t *testing.T) {
	title := " 身体測定 "
	allDay := true
	belongings := " 水筒、帽子 "
	deadline := "2026-06-10"
	start := "09:00"
	end := "10:00"
	candidate := Candidate{
		Title:     "仮タイトル",
		EventDate: "2026-06-12",
		StartTime: &start,
		EndTime:   &end,
		Status:    StatusDraft,
	}

	updated, err := candidate.Updated(Update{
		Title:              &title,
		IsAllDay:           &allDay,
		Belongings:         &belongings,
		SubmissionDeadline: &deadline,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "身体測定" || updated.StartTime != nil || updated.EndTime != nil {
		t.Fatalf("unexpected updated candidate: %#v", updated)
	}
	if updated.Belongings != "水筒、帽子" || updated.SubmissionDeadline == nil || *updated.SubmissionDeadline != deadline {
		t.Fatalf("unexpected important details: %#v", updated)
	}
	if updated.Status != StatusConfirmed {
		t.Fatalf("expected confirmed status, got %q", updated.Status)
	}
}

func TestCandidateUpdatedValidatesRules(t *testing.T) {
	if _, err := (Candidate{Status: StatusRegistered}).Updated(Update{}); !errors.Is(err, ErrNotEditable) {
		t.Fatalf("expected not editable error, got %v", err)
	}

	allDay := false
	start := "16:00"
	end := "15:00"
	if _, err := (Candidate{Status: StatusDraft}).Updated(Update{IsAllDay: &allDay, StartTime: &start, EndTime: &end}); !errors.Is(err, ErrInvalidTimedRange) {
		t.Fatalf("expected timed range error, got %v", err)
	}
}

func TestCandidateUpdatedKeepsDeletedStatus(t *testing.T) {
	title := "再登録する予定"
	updated, err := (Candidate{Status: StatusDeleted, IsAllDay: true}).Updated(Update{Title: &title})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != StatusDeleted {
		t.Fatalf("expected deleted status, got %q", updated.Status)
	}
}
