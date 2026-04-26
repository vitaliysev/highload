package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"marketplace/search/internal/domain"
)

const (
	cardPrefix   = "search:card:"
	searchPrefix = "search:query:"
)

type SearchCache struct {
	client *redis.Client
}

func NewSearchCache(client *redis.Client) *SearchCache {
	return &SearchCache{client: client}
}

// кэш карточки объявления
func (c *SearchCache) GetCard(ctx context.Context, id uuid.UUID) (*domain.ListingCard, error) {
	data, err := c.client.Get(ctx, cardPrefix+id.String()).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("redis get card: %w", err)
	}
	var card domain.ListingCard
	if err := json.Unmarshal(data, &card); err != nil {
		return nil, fmt.Errorf("redis unmarshal card: %w", err)
	}
	return &card, nil
}

func (c *SearchCache) SetCard(ctx context.Context, card *domain.ListingCard, ttl time.Duration) error {
	data, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("redis marshal card: %w", err)
	}
	return c.client.Set(ctx, cardPrefix+card.ID.String(), data, ttl).Err()
}

// кэш поискового запроса
func (c *SearchCache) GetSearch(ctx context.Context, key string) (*domain.SearchResult, error) {
	data, err := c.client.Get(ctx, searchPrefix+key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("redis get search: %w", err)
	}
	var result domain.SearchResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("redis unmarshal search: %w", err)
	}
	return &result, nil
}

func (c *SearchCache) SetSearch(ctx context.Context, key string, result *domain.SearchResult, ttl time.Duration) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("redis marshal search: %w", err)
	}
	return c.client.Set(ctx, searchPrefix+key, data, ttl).Err()
}
