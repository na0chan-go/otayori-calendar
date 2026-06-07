package extractedevent

import (
	"context"
	"errors"

	extractedeventdomain "github.com/na0chan-go/otayori-calendar/backend/internal/domain/extractedevent"
	extractedeventport "github.com/na0chan-go/otayori-calendar/backend/internal/port/extractedevent"
)

var (
	ErrCalendarEventCreateFailed = errors.New("failed to create google calendar event")
	ErrSaveRegistrationFailed    = errors.New("failed to save extracted event")
)

type Register struct {
	Calendar   extractedeventport.CalendarGateway
	Repository extractedeventport.RegistrationRepository
}

func (u Register) Execute(ctx context.Context, registration extractedeventdomain.Registration) error {
	if err := registration.Validate(); err != nil {
		return err
	}

	calendarEventID, err := u.Calendar.Create(ctx)
	if err != nil {
		_ = u.Repository.Save(ctx, registration.Failed())
		return ErrCalendarEventCreateFailed
	}

	registration, err = registration.Registered(calendarEventID)
	if err != nil {
		_ = u.Repository.Save(ctx, registration)
		return err
	}
	if err := u.Repository.Save(ctx, registration); err != nil {
		return ErrSaveRegistrationFailed
	}
	return nil
}
