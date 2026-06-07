package extractedevent

import (
	"context"
	"errors"
	"testing"

	extractedeventdomain "github.com/na0chan-go/otayori-calendar/backend/internal/domain/extractedevent"
)

type stateRepositoryStub struct {
	saved extractedeventdomain.State
	err   error
}

func (r *stateRepositoryStub) SaveState(_ context.Context, state extractedeventdomain.State) error {
	r.saved = state
	return r.err
}

func TestChangeStateActions(t *testing.T) {
	repository := &stateRepositoryStub{}
	usecase := ChangeState{Repository: repository}
	state := extractedeventdomain.State{Status: extractedeventdomain.StatusDraft, CalendarEventID: "stale-id"}

	if err := usecase.Confirm(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	if repository.saved.Status != extractedeventdomain.StatusConfirmed || repository.saved.CalendarEventID != "" {
		t.Fatalf("unexpected confirmed state: %#v", repository.saved)
	}

	if err := usecase.Ignore(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	if repository.saved.Status != extractedeventdomain.StatusIgnored {
		t.Fatalf("unexpected ignored state: %#v", repository.saved)
	}

	if err := usecase.Restore(context.Background(), repository.saved, extractedeventdomain.StatusIgnored, extractedeventdomain.StatusDraft); err != nil {
		t.Fatal(err)
	}
	if repository.saved.Status != extractedeventdomain.StatusDraft {
		t.Fatalf("unexpected restored state: %#v", repository.saved)
	}
}

func TestChangeStateDoesNotSaveInvalidTransition(t *testing.T) {
	repository := &stateRepositoryStub{}
	usecase := ChangeState{Repository: repository}

	err := usecase.Ignore(context.Background(), extractedeventdomain.State{Status: extractedeventdomain.StatusRegistered})
	if !errors.Is(err, extractedeventdomain.ErrRegisteredCannotBeIgnored) {
		t.Fatalf("expected ignored error, got %v", err)
	}
	if repository.saved.Status != "" {
		t.Fatal("invalid transition must not be saved")
	}
}

func TestChangeStateReturnsSaveError(t *testing.T) {
	usecase := ChangeState{Repository: &stateRepositoryStub{err: errors.New("db unavailable")}}
	err := usecase.Confirm(context.Background(), extractedeventdomain.State{Status: extractedeventdomain.StatusDraft})
	if !errors.Is(err, ErrSaveStateFailed) {
		t.Fatalf("expected save error, got %v", err)
	}
}
