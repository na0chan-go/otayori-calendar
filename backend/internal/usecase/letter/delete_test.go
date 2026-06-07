package letter

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	letterdomain "github.com/na0chan-go/otayori-calendar/backend/internal/domain/letter"
	letterport "github.com/na0chan-go/otayori-calendar/backend/internal/port/letter"
)

func TestDeleteExecute(t *testing.T) {
	target := letterdomain.Letter{ID: uuid.New(), UserID: uuid.New(), ImagePath: "letter.png"}
	repository := &deletionRepositoryStub{letter: target}
	storage := &imageStorageStub{quarantinePath: "letter.png.deleting"}

	err := (Delete{Repository: repository, Storage: storage}).Execute(context.Background(), target.ID, target.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if !repository.deleted || !storage.removed {
		t.Fatalf("expected delete and remove: repository=%#v storage=%#v", repository, storage)
	}
}

func TestDeleteExecuteRestoresImageWhenRepositoryDeleteFails(t *testing.T) {
	target := letterdomain.Letter{ID: uuid.New(), UserID: uuid.New(), ImagePath: "letter.png"}
	repository := &deletionRepositoryStub{letter: target, deleteErr: errors.New("database failed")}
	storage := &imageStorageStub{quarantinePath: "letter.png.deleting"}

	err := (Delete{Repository: repository, Storage: storage}).Execute(context.Background(), target.ID, target.UserID)
	if !errors.Is(err, ErrDeleteFailed) {
		t.Fatalf("expected delete error, got %v", err)
	}
	if !storage.restored {
		t.Fatal("expected quarantined image to be restored")
	}
}

func TestDeleteExecuteMapsNotFound(t *testing.T) {
	repository := &deletionRepositoryStub{findErr: letterport.ErrNotFound}
	err := (Delete{Repository: repository, Storage: &imageStorageStub{}}).
		Execute(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

type deletionRepositoryStub struct {
	letter    letterdomain.Letter
	findErr   error
	deleteErr error
	deleted   bool
}

func (r *deletionRepositoryStub) FindOwned(context.Context, uuid.UUID, uuid.UUID) (letterdomain.Letter, error) {
	return r.letter, r.findErr
}

func (r *deletionRepositoryStub) Delete(context.Context, letterdomain.Letter) error {
	r.deleted = true
	return r.deleteErr
}

type imageStorageStub struct {
	quarantinePath string
	quarantineErr  error
	restoreErr     error
	removeErr      error
	restored       bool
	removed        bool
}

func (s *imageStorageStub) Quarantine(context.Context, string) (string, error) {
	return s.quarantinePath, s.quarantineErr
}

func (s *imageStorageStub) Restore(context.Context, string, string) error {
	s.restored = true
	return s.restoreErr
}

func (s *imageStorageStub) Remove(context.Context, string) error {
	s.removed = true
	return s.removeErr
}
