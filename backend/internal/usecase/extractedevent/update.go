package extractedevent

import (
	"context"
	"errors"

	extractedeventdomain "github.com/na0chan-go/otayori-calendar/backend/internal/domain/extractedevent"
	extractedeventport "github.com/na0chan-go/otayori-calendar/backend/internal/port/extractedevent"
)

var ErrSaveCandidateFailed = errors.New("failed to update extracted event")

type Update struct {
	Repository extractedeventport.CandidateRepository
}

func (u Update) Execute(ctx context.Context, candidate extractedeventdomain.Candidate, changes extractedeventdomain.Update) error {
	updated, err := candidate.Updated(changes)
	if err != nil {
		return err
	}
	if err := u.Repository.SaveCandidate(ctx, updated); err != nil {
		return ErrSaveCandidateFailed
	}
	return nil
}
