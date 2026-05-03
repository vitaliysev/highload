package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ModerationTask struct {
	ListingID uuid.UUID `json:"listing_id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type ModerationStatus string

const (
	ModerationApproved ModerationStatus = "published"
	ModerationRejected ModerationStatus = "rejected"
)

type ListingRepository interface {
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
}
