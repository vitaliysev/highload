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
	ID            uuid.UUID     `json:"id"`
	UserID        uuid.UUID     `json:"user_id"`
	Title         string        `json:"title"`
	Description   string        `json:"description"`
	Price         float64       `json:"price"`
	Category      string        `json:"category"`
	Location      string        `json:"location"`
	Status        ListingStatus `json:"status"`
	IsPromoted    bool          `json:"is_promoted"`
	PromotedUntil *time.Time    `json:"promoted_until,omitempty"`
	ViewsCount    int           `json:"views_count"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

type ModerationTask struct {
	ListingID uuid.UUID `json:"listing_id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type ListingCard struct {
	Listing
	SellerName  string `json:"seller_name"`
	SellerPhone string `json:"seller_phone"`
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
