package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID
	Name      string
	Email     string
	Phone     string
	City      string
	AvatarURL string
	CreatedAt time.Time
}

type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
}
