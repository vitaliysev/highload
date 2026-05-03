package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"marketplace/search/internal/domain"
)

const (
	normalCacheTTL   = 30 * time.Second
	degradedCacheTTL = 10 * time.Second
)

type SearchService struct {
	repo  domain.SearchRepository
	cache domain.SearchCache
}

func New(repo domain.SearchRepository, cache domain.SearchCache) *SearchService {
	return &SearchService{repo: repo, cache: cache}
}

// Cache-Aside: сначала Redis, при промахе — PostgreSQL FTS.
// TTL кэша: 30s в нормальном режиме, 10s при деградации
// X-Search-Degraded заголовок выставляется в handler на основе result.Degraded.
func (s *SearchService) Search(ctx context.Context, q domain.SearchQuery) (*domain.SearchResult, error) {
	key := buildCacheKey(q)

	if result, err := s.cache.GetSearch(ctx, key); err == nil {
		return result, nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		log.Printf("warn: cache get search: %v", err)
	}

	result, err := s.repo.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	ttl := normalCacheTTL
	if result.Degraded {
		ttl = degradedCacheTTL
	}

	if err := s.cache.SetSearch(ctx, key, result, ttl); err != nil {
		log.Printf("warn: cache set search: %v", err)
	}

	return result, nil
}

func buildCacheKey(q domain.SearchQuery) string {
	raw := fmt.Sprintf("%s|%s|%v|%v|%s|%d|%d",
		q.Query, q.Category, q.PriceMin, q.PriceMax, q.Location, q.Limit, q.Offset)
	sum := md5.Sum([]byte(raw))
	return hex.EncodeToString(sum[:])
}
