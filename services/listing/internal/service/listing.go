package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"marketplace/listing/internal/domain"
)

const cardCacheTTL = 5 * time.Minute

type ListingService struct {
	repo      domain.ListingRepository
	cache     domain.ListingCache
	modPub    domain.ModerationPublisher
	promoPub  domain.PromotionPublisher
}

func New(
	repo domain.ListingRepository,
	cache domain.ListingCache,
	modPub domain.ModerationPublisher,
	promoPub domain.PromotionPublisher,
) *ListingService {
	return &ListingService{repo: repo, cache: cache, modPub: modPub, promoPub: promoPub}
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
	if err := s.modPub.Publish(ctx, task); err != nil {
		log.Printf("warn: failed to publish moderation task for %s: %v", created.ID, err)
	}

	return created, nil
}

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

func (s *ListingService) GetUploadURL(ctx context.Context, userID, listingID uuid.UUID, filename, contentType string, sizeBytes int64) (*domain.Photo, string, error) {
	card, err := s.repo.GetCardByID(ctx, listingID)
	if err != nil {
		return nil, "", err
	}
	if card.UserID != userID {
		return nil, "", domain.ErrForbidden
	}

	count, err := s.repo.CountPhotos(ctx, listingID)
	if err != nil {
		return nil, "", err
	}
	if count >= 10 {
		return nil, "", domain.ErrPhotoLimitReached
	}

	storageKey := fmt.Sprintf("listings/%s/%s", listingID, filename)
	photo, err := s.repo.CreatePhoto(ctx, listingID, storageKey, int16(count))
	if err != nil {
		return nil, "", err
	}

	fakeURL := fmt.Sprintf("https://storage.example.com/%s?X-Amz-Expires=900&photo_id=%s", storageKey, photo.ID)
	return photo, fakeURL, nil
}

func (s *ListingService) Promote(ctx context.Context, userID, listingID uuid.UUID, plan, paymentMethod string) (*domain.Payment, *domain.Promotion, error) {
	card, err := s.repo.GetCardByID(ctx, listingID)
	if err != nil {
		return nil, nil, err
	}
	if card.UserID != userID {
		return nil, nil, domain.ErrForbidden
	}
	if card.Status != domain.StatusPublished {
		return nil, nil, domain.ErrBadRequest
	}

	duration, ok := domain.PlanDurations[plan]
	if !ok {
		return nil, nil, domain.ErrBadRequest
	}
	amount, _ := domain.PlanPrices[plan]

	active, err := s.repo.HasActivePromotion(ctx, listingID)
	if err != nil {
		return nil, nil, err
	}
	if active {
		return nil, nil, domain.ErrConflict
	}

	payment := &domain.Payment{
		ListingID:         listingID,
		UserID:            userID,
		ExternalPaymentID: uuid.New().String(),
		Amount:            amount,
		Currency:          "RUB",
		Status:            "paid",
		PaymentMethod:     paymentMethod,
	}
	expiresAt := time.Now().Add(duration)

	p, promo, err := s.repo.CreatePaymentAndPromotion(ctx, payment, plan, expiresAt)
	if err != nil {
		return nil, nil, err
	}

	event := domain.PromotionActivatedEvent{
		ListingID: listingID,
		ExpiresAt: expiresAt,
	}
	if err := s.promoPub.PublishPromotionActivated(ctx, event); err != nil {
		log.Printf("warn: failed to publish promotion-activated for %s: %v", listingID, err)
	}

	return p, promo, nil
}

func (s *ListingService) GetUserListings(ctx context.Context, userID uuid.UUID, f domain.UserListingsFilter) ([]*domain.Listing, int, error) {
	return s.repo.GetUserListings(ctx, userID, f)
}
