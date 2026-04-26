package service

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"

	"marketplace/listing/internal/domain"
)

const cardCacheTTL = 5 * time.Minute

type ListingService struct {
	repo      domain.ListingRepository
	cache     domain.ListingCache
	publisher domain.ModerationPublisher
}

func New(repo domain.ListingRepository, cache domain.ListingCache, publisher domain.ModerationPublisher) *ListingService {
	return &ListingService{repo: repo, cache: cache, publisher: publisher}
}

func (s *ListingService) Create(ctx context.Context, l *domain.Listing) (*domain.Listing, error) {
	if l.Title == "" || l.Category == "" || l.Location == "" {
		return nil, domain.ErrBadRequest
	}

	created, err := s.repo.Create(ctx, l)
	if err != nil {
		return nil, err
	}

	task := domain.ModerationTask{
		ListingID: created.ID,
		Title:     created.Title,
		CreatedAt: created.CreatedAt,
	}
	if err := s.publisher.Publish(ctx, task); err != nil {
		log.Printf("warn: failed to publish moderation task for %s: %v", created.ID, err)
	}

	return created, nil
}

// реализует Cache-Aside: сначала Redis, при промахе — PostgreSQL.
// кэш инвалидируется при получении promotion-activated.
func (s *ListingService) GetCard(ctx context.Context, id uuid.UUID) (*domain.ListingCard, error) {
	card, err := s.cache.GetCard(ctx, id)
	if err == nil {
		return card, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		log.Printf("warn: cache get card %s: %v", id, err)
	}

	card, err = s.repo.GetCardByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.cache.SetCard(ctx, card, cardCacheTTL); err != nil {
		log.Printf("warn: cache set card %s: %v", id, err)
	}

	return card, nil
}
