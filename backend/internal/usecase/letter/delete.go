package letter

import (
	"context"
	"errors"

	"github.com/google/uuid"
	letterport "github.com/na0chan-go/otayori-calendar/backend/internal/port/letter"
)

var (
	ErrNotFound              = errors.New("letter not found")
	ErrLoadFailed            = errors.New("failed to load letter")
	ErrPrepareDeletionFailed = errors.New("failed to prepare letter image deletion")
	ErrDeleteFailed          = errors.New("failed to delete letter")
	ErrImageDeletionFailed   = errors.New("failed to delete letter image")
)

type Delete struct {
	Repository letterport.DeletionRepository
	Storage    letterport.ImageStorage
}

func (u Delete) Execute(ctx context.Context, letterID, userID uuid.UUID) error {
	target, err := u.Repository.FindOwned(ctx, letterID, userID)
	if errors.Is(err, letterport.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return ErrLoadFailed
	}

	quarantinePath, err := u.Storage.Quarantine(ctx, target.ImagePath)
	if err != nil {
		return ErrPrepareDeletionFailed
	}
	if err := u.Repository.Delete(ctx, target); err != nil {
		_ = u.Storage.Restore(ctx, quarantinePath, target.ImagePath)
		return ErrDeleteFailed
	}
	if err := u.Storage.Remove(ctx, quarantinePath); err != nil {
		return ErrImageDeletionFailed
	}
	return nil
}
