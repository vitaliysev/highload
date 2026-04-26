package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"marketplace/listing/internal/domain"
)

type ListingRepo struct {
	db *pgxpool.Pool
}

func NewListingRepo(db *pgxpool.Pool) *ListingRepo {
	return &ListingRepo{db: db}
}

func (r *ListingRepo) Create(ctx context.Context, l *domain.Listing) (*domain.Listing, error) {
	const q = `
		INSERT INTO listings (user_id, title, description, price, category, location)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, title, description, price, category, location,
		          status, is_promoted, promoted_until, views_count, created_at, updated_at`

	var out domain.Listing
	err := r.db.QueryRow(ctx, q,
		l.UserID, l.Title, l.Description, l.Price, l.Category, l.Location,
	).Scan(
		&out.ID, &out.UserID, &out.Title, &out.Description, &out.Price,
		&out.Category, &out.Location, &out.Status, &out.IsPromoted,
		&out.PromotedUntil, &out.ViewsCount, &out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// метод кэшируется в Redis
func (r *ListingRepo) GetCardByID(ctx context.Context, id uuid.UUID) (*domain.ListingCard, error) {
	const q = `
		SELECT
			l.id, l.user_id, l.title, l.description, l.price, l.category, l.location,
			l.status, l.is_promoted, l.promoted_until, l.views_count, l.created_at, l.updated_at,
			u.name, COALESCE(u.phone, '') AS phone
		FROM listings l
		JOIN users u ON u.id = l.user_id
		WHERE l.id = $1`

	var c domain.ListingCard
	err := r.db.QueryRow(ctx, q, id).Scan(
		&c.ID, &c.UserID, &c.Title, &c.Description, &c.Price,
		&c.Category, &c.Location, &c.Status, &c.IsPromoted,
		&c.PromotedUntil, &c.ViewsCount, &c.CreatedAt, &c.UpdatedAt,
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

func (r *ListingRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ListingStatus) error {
	const q = `UPDATE listings SET status = $1 WHERE id = $2`
	tag, err := r.db.Exec(ctx, q, string(status), id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
