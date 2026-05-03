package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"marketplace/listing/internal/domain"
)

const cardPrefix = "listing:card:"

type ListingCache struct {
	client *redis.Client
}

func NewListingCache(client *redis.Client) *ListingCache {
	return &ListingCache{client: client}
}

func cardKey(id uuid.UUID) string {
	return cardPrefix + id.String()
}

func (c *ListingCache) GetCard(ctx context.Context, id uuid.UUID) (*domain.ListingCard, error) {
	data, err := c.client.Get(ctx, cardKey(id)).Bytes()
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

// кэширует карточку
func (c *ListingCache) SetCard(ctx context.Context, card *domain.ListingCard, ttl time.Duration) error {
	data, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("redis marshal card: %w", err)
	}
	return c.client.Set(ctx, cardKey(card.ID), data, ttl).Err()
}

// вызывается при promotion-activated
func (c *ListingCache) Delete(ctx context.Context, id uuid.UUID) error {
	return c.client.Del(ctx, cardKey(id)).Err()
}
