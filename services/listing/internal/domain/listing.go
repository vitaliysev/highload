package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ListingStatus string

const (
	StatusPending   ListingStatus = "pending"
	StatusPublished ListingStatus = "published"
	StatusRejected  ListingStatus = "rejected"
	StatusArchived  ListingStatus = "archived"
)

type Listing struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	Title         string
	Description   string
	Price         float64
	Category      string
	Location      string
	Status        ListingStatus
	IsPromoted    bool
	PromotedUntil *time.Time
	ViewsCount    int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ModerationTask struct {
	ListingID uuid.UUID `json:"listing_id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type ListingCard struct {
	Listing
	SellerName  string
	SellerPhone string
}

type ListingRepository interface {
	Create(ctx context.Context, l *Listing) (*Listing, error)
	GetCardByID(ctx context.Context, id uuid.UUID) (*ListingCard, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status ListingStatus) error
}

// Cache-Aside для карточек объявлений
type ListingCache interface {
	GetCard(ctx context.Context, id uuid.UUID) (*ListingCard, error)
	SetCard(ctx context.Context, card *ListingCard, ttl time.Duration) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type ModerationPublisher interface {
	Publish(ctx context.Context, task ModerationTask) error
}

type PromotionActivatedEvent struct {
	ListingID uuid.UUID `json:"listing_id"`
	ExpiresAt time.Time `json:"expires_at"`
}
