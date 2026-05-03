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

type Photo struct {
	ID         uuid.UUID `json:"id"`
	ListingID  uuid.UUID `json:"listing_id"`
	StorageKey string    `json:"storage_key"`
	Position   int16     `json:"position"`
	CreatedAt  time.Time `json:"created_at"`
}

type Payment struct {
	ID                uuid.UUID `json:"id"`
	ListingID         uuid.UUID `json:"listing_id"`
	UserID            uuid.UUID `json:"user_id"`
	ExternalPaymentID string    `json:"external_payment_id"`
	Amount            int64     `json:"amount"`
	Currency          string    `json:"currency"`
	Status            string    `json:"status"`
	PaymentMethod     string    `json:"payment_method"`
	CreatedAt         time.Time `json:"created_at"`
}

type Promotion struct {
	ID        uuid.UUID `json:"id"`
	ListingID uuid.UUID `json:"listing_id"`
	PaymentID uuid.UUID `json:"payment_id"`
	Plan      string    `json:"plan"`
	StartsAt  time.Time `json:"starts_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Active    bool      `json:"active"`
}

var PlanDurations = map[string]time.Duration{
	"top_7days":  7 * 24 * time.Hour,
	"top_30days": 30 * 24 * time.Hour,
}

var PlanPrices = map[string]int64{
	"top_7days":  14900,
	"top_30days": 49900,
}

type UserListingsFilter struct {
	Status  *ListingStatus
	Page    int
	PerPage int
}

type ListingRepository interface {
	Create(ctx context.Context, l *Listing) (*Listing, error)
	GetCardByID(ctx context.Context, id uuid.UUID) (*ListingCard, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status ListingStatus) error
	CreatePhoto(ctx context.Context, listingID uuid.UUID, storageKey string, position int16) (*Photo, error)
	CountPhotos(ctx context.Context, listingID uuid.UUID) (int, error)
	HasActivePromotion(ctx context.Context, listingID uuid.UUID) (bool, error)
	CreatePaymentAndPromotion(ctx context.Context, payment *Payment, plan string, expiresAt time.Time) (*Payment, *Promotion, error)
	GetUserListings(ctx context.Context, userID uuid.UUID, f UserListingsFilter) ([]*Listing, int, error)
}

type ListingCache interface {
	GetCard(ctx context.Context, id uuid.UUID) (*ListingCard, error)
	SetCard(ctx context.Context, card *ListingCard, ttl time.Duration) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type ModerationPublisher interface {
	Publish(ctx context.Context, task ModerationTask) error
}

type PromotionPublisher interface {
	PublishPromotionActivated(ctx context.Context, event PromotionActivatedEvent) error
}

type PromotionActivatedEvent struct {
	ListingID uuid.UUID `json:"listing_id"`
	ExpiresAt time.Time `json:"expires_at"`
}
