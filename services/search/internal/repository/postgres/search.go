package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"marketplace/search/internal/domain"
)

type SearchRepo struct {
	db *pgxpool.Pool
}

func NewSearchRepo(db *pgxpool.Pool) *SearchRepo {
	return &SearchRepo{db: db}
}

// search — PostgreSQL FTS через search_vector (GIN-индекс).
// это fallback-режим архитектуры: в prod основной бэкенд — Elasticsearch.
// порядок: продвинутые объявления сначала, затем по дате
func (r *SearchRepo) Search(ctx context.Context, q domain.SearchQuery) (*domain.SearchResult, error) {
	args := []any{}
	conditions := []string{"l.status = 'published'"}
	idx := 1

	if q.Query != "" {
		conditions = append(conditions,
			fmt.Sprintf("l.search_vector @@ plainto_tsquery('russian', $%d)", idx))
		args = append(args, q.Query)
		idx++
	}
	if q.Category != "" {
		conditions = append(conditions, fmt.Sprintf("l.category = $%d", idx))
		args = append(args, q.Category)
		idx++
	}
	if q.PriceMin != nil {
		conditions = append(conditions, fmt.Sprintf("l.price >= $%d", idx))
		args = append(args, *q.PriceMin)
		idx++
	}
	if q.PriceMax != nil {
		conditions = append(conditions, fmt.Sprintf("l.price <= $%d", idx))
		args = append(args, *q.PriceMax)
		idx++
	}
	if q.Location != "" {
		conditions = append(conditions, fmt.Sprintf("l.location ILIKE $%d", idx))
		args = append(args, "%"+q.Location+"%")
		idx++
	}

	where := strings.Join(conditions, " AND ")

	limit := q.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := q.Offset
	if offset < 0 {
		offset = 0
	}

	// запрос данных + данных продавца (один JOIN вместо N+1).
	dataQ := fmt.Sprintf(`
		SELECT
			l.id, l.user_id, l.title, l.description, l.price,
			l.category, l.location, l.is_promoted, l.views_count, l.created_at,
			u.name, COALESCE(u.phone, '')
		FROM listings l
		JOIN users u ON u.id = l.user_id
		WHERE %s
		ORDER BY l.is_promoted DESC, l.created_at DESC
		LIMIT $%d OFFSET $%d`, where, idx, idx+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, dataQ, args...)
	if err != nil {
		return nil, fmt.Errorf("search query: %w", err)
	}
	defer rows.Close()

	var items []*domain.ListingCard
	for rows.Next() {
		var c domain.ListingCard
		if err := rows.Scan(
			&c.ID, &c.UserID, &c.Title, &c.Description, &c.Price,
			&c.Category, &c.Location, &c.IsPromoted, &c.ViewsCount, &c.CreatedAt,
			&c.SellerName, &c.SellerPhone,
		); err != nil {
			return nil, fmt.Errorf("search scan: %w", err)
		}
		items = append(items, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// пагинация
	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM listings l WHERE %s`, where)
	var total int
	if err := r.db.QueryRow(ctx, countQ, args[:idx-1]...).Scan(&total); err != nil {
		return nil, fmt.Errorf("search count: %w", err)
	}

	return &domain.SearchResult{
		Items:    items,
		Total:    total,
		Degraded: true, // всегда true, пока используем PG FTS
	}, nil
}

func (r *SearchRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.ListingCard, error) {
	const q = `
		SELECT
			l.id, l.user_id, l.title, l.description, l.price,
			l.category, l.location, l.is_promoted, l.views_count, l.created_at,
			u.name, COALESCE(u.phone, '')
		FROM listings l
		JOIN users u ON u.id = l.user_id
		WHERE l.id = $1`

	var c domain.ListingCard
	err := r.db.QueryRow(ctx, q, id).Scan(
		&c.ID, &c.UserID, &c.Title, &c.Description, &c.Price,
		&c.Category, &c.Location, &c.IsPromoted, &c.ViewsCount, &c.CreatedAt,
		&c.SellerName, &c.SellerPhone,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}
