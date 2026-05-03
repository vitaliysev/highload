package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ListingCard struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	SellerName  string    `json:"seller_name"`
	SellerPhone string    `json:"seller_phone"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Category    string    `json:"category"`
	Location    string    `json:"location"`
	IsPromoted  bool      `json:"is_promoted"`
	ViewsCount  int       `json:"views_count"`
	CreatedAt   time.Time `json:"created_at"`
}

type SearchQuery struct {
	Query    string
	Category string
	PriceMin *float64
	PriceMax *float64
	Location string
	Limit    int
	Offset   int
}

type SearchResult struct {
	Items    []*ListingCard `json:"items"`
	Total    int            `json:"total"`
	Degraded bool           `json:"degraded"`
}

type SearchRepository interface {
	Search(ctx context.Context, q SearchQuery) (*SearchResult, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ListingCard, error)
}

type SearchCache interface {
	GetCard(ctx context.Context, id uuid.UUID) (*ListingCard, error)
	SetCard(ctx context.Context, card *ListingCard, ttl time.Duration) error
	GetSearch(ctx context.Context, key string) (*SearchResult, error)
	SetSearch(ctx context.Context, key string, result *SearchResult, ttl time.Duration) error
}
