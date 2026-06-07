package extractedevent

import (
	"context"
	"errors"
	"testing"

	extractedeventdomain "github.com/na0chan-go/otayori-calendar/backend/internal/domain/extractedevent"
)

type candidateRepositoryStub struct {
	saved extractedeventdomain.Candidate
	err   error
}

func (r *candidateRepositoryStub) SaveCandidate(_ context.Context, candidate extractedeventdomain.Candidate) error {
	r.saved = candidate
	return r.err
}

func TestUpdateExecuteSavesUpdatedCandidate(t *testing.T) {
	title := "身体測定"
	repository := &candidateRepositoryStub{}
	usecase := Update{Repository: repository}

	err := usecase.Execute(
		context.Background(),
		extractedeventdomain.Candidate{Status: extractedeventdomain.StatusDraft, IsAllDay: true},
		extractedeventdomain.Update{Title: &title},
	)
	if err != nil {
		t.Fatal(err)
	}
	if repository.saved.Title != title || repository.saved.Status != extractedeventdomain.StatusConfirmed {
		t.Fatalf("unexpected saved candidate: %#v", repository.saved)
	}
}

func TestUpdateExecuteDoesNotSaveInvalidCandidate(t *testing.T) {
	repository := &candidateRepositoryStub{}
	usecase := Update{Repository: repository}

	err := usecase.Execute(
		context.Background(),
		extractedeventdomain.Candidate{Status: extractedeventdomain.StatusRegistered},
		extractedeventdomain.Update{},
	)
	if !errors.Is(err, extractedeventdomain.ErrNotEditable) {
		t.Fatalf("expected not editable error, got %v", err)
	}
	if repository.saved.Status != "" {
		t.Fatal("invalid candidate must not be saved")
	}
}

func TestUpdateExecuteReturnsSaveError(t *testing.T) {
	usecase := Update{Repository: &candidateRepositoryStub{err: errors.New("db unavailable")}}
	err := usecase.Execute(
		context.Background(),
		extractedeventdomain.Candidate{Status: extractedeventdomain.StatusDraft, IsAllDay: true},
		extractedeventdomain.Update{},
	)
	if !errors.Is(err, ErrSaveCandidateFailed) {
		t.Fatalf("expected save error, got %v", err)
	}
}
