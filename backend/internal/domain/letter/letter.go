package letter

import "github.com/google/uuid"

type Letter struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ImagePath string
}
