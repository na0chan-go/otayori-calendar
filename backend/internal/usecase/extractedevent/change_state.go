package extractedevent

import (
	"context"
	"errors"

	extractedeventdomain "github.com/na0chan-go/otayori-calendar/backend/internal/domain/extractedevent"
	extractedeventport "github.com/na0chan-go/otayori-calendar/backend/internal/port/extractedevent"
)

var ErrSaveStateFailed = errors.New("failed to save extracted event state")

type ChangeState struct {
	Repository extractedeventport.StateRepository
}

func (u ChangeState) Confirm(ctx context.Context, state extractedeventdomain.State) error {
	next, err := state.Confirmed()
	return u.save(ctx, next, err)
}

func (u ChangeState) Ignore(ctx context.Context, state extractedeventdomain.State) error {
	next, err := state.Ignored()
	return u.save(ctx, next, err)
}

func (u ChangeState) Restore(ctx context.Context, state extractedeventdomain.State, expectedStatus, targetStatus string) error {
	next, err := state.Restored(expectedStatus, targetStatus)
	return u.save(ctx, next, err)
}

func (u ChangeState) save(ctx context.Context, state extractedeventdomain.State, transitionErr error) error {
	if transitionErr != nil {
		return transitionErr
	}
	if err := u.Repository.SaveState(ctx, state); err != nil {
		return ErrSaveStateFailed
	}
	return nil
}
