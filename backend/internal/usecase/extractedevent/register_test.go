package extractedevent

import (
	"context"
	"errors"
	"testing"

	extractedeventdomain "github.com/na0chan-go/otayori-calendar/backend/internal/domain/extractedevent"
)

type calendarGatewayStub struct {
	eventID string
	err     error
	called  bool
}

func (g *calendarGatewayStub) Create(context.Context) (string, error) {
	g.called = true
	return g.eventID, g.err
}

type registrationRepositoryStub struct {
	saved extractedeventdomain.Registration
	err   error
}

func (r *registrationRepositoryStub) Save(_ context.Context, registration extractedeventdomain.Registration) error {
	r.saved = registration
	return r.err
}

func TestRegisterExecuteSavesRegisteredState(t *testing.T) {
	calendar := &calendarGatewayStub{eventID: "event-id"}
	repository := &registrationRepositoryStub{}
	usecase := Register{Calendar: calendar, Repository: repository}

	if err := usecase.Execute(context.Background(), extractedeventdomain.Registration{Status: extractedeventdomain.StatusConfirmed}); err != nil {
		t.Fatal(err)
	}
	if repository.saved.Status != extractedeventdomain.StatusRegistered || repository.saved.CalendarEventID != "event-id" {
		t.Fatalf("unexpected saved state: %#v", repository.saved)
	}
}

func TestRegisterExecuteSavesFailedStateWhenCalendarFails(t *testing.T) {
	calendar := &calendarGatewayStub{err: errors.New("calendar unavailable")}
	repository := &registrationRepositoryStub{}
	usecase := Register{Calendar: calendar, Repository: repository}

	err := usecase.Execute(context.Background(), extractedeventdomain.Registration{Status: extractedeventdomain.StatusConfirmed})
	if !errors.Is(err, ErrCalendarEventCreateFailed) {
		t.Fatalf("expected calendar error, got %v", err)
	}
	if repository.saved.Status != extractedeventdomain.StatusFailed {
		t.Fatalf("expected failed state, got %#v", repository.saved)
	}
}

func TestRegisterExecuteValidatesBeforeCallingCalendar(t *testing.T) {
	calendar := &calendarGatewayStub{eventID: "event-id"}
	usecase := Register{Calendar: calendar, Repository: &registrationRepositoryStub{}}

	err := usecase.Execute(context.Background(), extractedeventdomain.Registration{Status: "draft"})
	if !errors.Is(err, extractedeventdomain.ErrNotRegisterable) {
		t.Fatalf("expected not registerable error, got %v", err)
	}
	if calendar.called {
		t.Fatal("calendar must not be called for invalid registration")
	}
}

func TestRegisterExecuteReturnsSaveError(t *testing.T) {
	usecase := Register{
		Calendar:   &calendarGatewayStub{eventID: "event-id"},
		Repository: &registrationRepositoryStub{err: errors.New("db unavailable")},
	}

	err := usecase.Execute(context.Background(), extractedeventdomain.Registration{Status: extractedeventdomain.StatusConfirmed})
	if !errors.Is(err, ErrSaveRegistrationFailed) {
		t.Fatalf("expected save error, got %v", err)
	}
}
