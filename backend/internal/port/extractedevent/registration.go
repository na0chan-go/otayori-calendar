package extractedevent

import (
	"context"

	extractedeventdomain "github.com/na0chan-go/otayori-calendar/backend/internal/domain/extractedevent"
)

type CalendarGateway interface {
	Create(ctx context.Context) (string, error)
}

type RegistrationRepository interface {
	Save(ctx context.Context, registration extractedeventdomain.Registration) error
}

type StateRepository interface {
	SaveState(ctx context.Context, state extractedeventdomain.State) error
}

type CandidateRepository interface {
	SaveCandidate(ctx context.Context, candidate extractedeventdomain.Candidate) error
}
