package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ListingCard struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	SellerName  string
	SellerPhone string
	Title       string
	Description string
	Price       float64
	Category    string
	Location    string
	IsPromoted  bool
	ViewsCount  int
	CreatedAt   time.Time
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
	Items    []*ListingCard
	Total    int
	Degraded bool // true — поиск через PG FTS fallback, а не Elasticsearch
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
