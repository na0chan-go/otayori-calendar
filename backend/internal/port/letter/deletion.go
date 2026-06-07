package letter

import (
	"context"
	"errors"

	"github.com/google/uuid"
	letterdomain "github.com/na0chan-go/otayori-calendar/backend/internal/domain/letter"
)

var ErrNotFound = errors.New("letter not found")

type DeletionRepository interface {
	FindOwned(ctx context.Context, letterID, userID uuid.UUID) (letterdomain.Letter, error)
	Delete(ctx context.Context, letter letterdomain.Letter) error
}

type ImageStorage interface {
	Quarantine(ctx context.Context, imagePath string) (string, error)
	Restore(ctx context.Context, quarantinePath, imagePath string) error
	Remove(ctx context.Context, quarantinePath string) error
}
