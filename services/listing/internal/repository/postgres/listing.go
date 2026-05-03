package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

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

func (r *ListingRepo) CreatePhoto(ctx context.Context, listingID uuid.UUID, storageKey string, position int16) (*domain.Photo, error) {
	const q = `
		INSERT INTO listing_photos (listing_id, storage_key, position)
		VALUES ($1, $2, $3)
		RETURNING id, listing_id, storage_key, position, created_at`

	var p domain.Photo
	err := r.db.QueryRow(ctx, q, listingID, storageKey, position).Scan(
		&p.ID, &p.ListingID, &p.StorageKey, &p.Position, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ListingRepo) CountPhotos(ctx context.Context, listingID uuid.UUID) (int, error) {
	const q = `SELECT COUNT(*) FROM listing_photos WHERE listing_id = $1`
	var n int
	if err := r.db.QueryRow(ctx, q, listingID).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func (r *ListingRepo) HasActivePromotion(ctx context.Context, listingID uuid.UUID) (bool, error) {
	const q = `
		SELECT EXISTS(
			SELECT 1 FROM promotions
			WHERE listing_id = $1 AND active = true AND expires_at > now()
		)`
	var exists bool
	if err := r.db.QueryRow(ctx, q, listingID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (r *ListingRepo) CreatePaymentAndPromotion(ctx context.Context, payment *domain.Payment, plan string, expiresAt time.Time) (*domain.Payment, *domain.Promotion, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const qPayment = `
		INSERT INTO payments (listing_id, user_id, external_payment_id, amount, currency, status, payment_method)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, listing_id, user_id, external_payment_id, amount, currency, status, payment_method, created_at`

	var p domain.Payment
	err = tx.QueryRow(ctx, qPayment,
		payment.ListingID, payment.UserID, payment.ExternalPaymentID,
		payment.Amount, payment.Currency, payment.Status, payment.PaymentMethod,
	).Scan(
		&p.ID, &p.ListingID, &p.UserID, &p.ExternalPaymentID,
		&p.Amount, &p.Currency, &p.Status, &p.PaymentMethod, &p.CreatedAt,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("insert payment: %w", err)
	}

	const qPromotion = `
		INSERT INTO promotions (listing_id, payment_id, plan, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, listing_id, payment_id, plan, starts_at, expires_at, active`

	var promo domain.Promotion
	err = tx.QueryRow(ctx, qPromotion, payment.ListingID, p.ID, plan, expiresAt).Scan(
		&promo.ID, &promo.ListingID, &promo.PaymentID,
		&promo.Plan, &promo.StartsAt, &promo.ExpiresAt, &promo.Active,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("insert promotion: %w", err)
	}

	const qListing = `UPDATE listings SET is_promoted = true, promoted_until = $1 WHERE id = $2`
	if _, err = tx.Exec(ctx, qListing, expiresAt, payment.ListingID); err != nil {
		return nil, nil, fmt.Errorf("update listing promotion: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit tx: %w", err)
	}
	return &p, &promo, nil
}

func (r *ListingRepo) GetUserListings(ctx context.Context, userID uuid.UUID, f domain.UserListingsFilter) ([]*domain.Listing, int, error) {
	if f.PerPage <= 0 {
		f.PerPage = 20
	}
	if f.Page <= 0 {
		f.Page = 1
	}
	offset := (f.Page - 1) * f.PerPage

	const qCount = `
		SELECT COUNT(*) FROM listings
		WHERE user_id = $1 AND ($2::listing_status IS NULL OR status = $2)`

	var total int
	if err := r.db.QueryRow(ctx, qCount, userID, f.Status).Scan(&total); err != nil {
		return nil, 0, err
	}

	const qRows = `
		SELECT id, user_id, title, description, price, category, location,
		       status, is_promoted, promoted_until, views_count, created_at, updated_at
		FROM listings
		WHERE user_id = $1 AND ($2::listing_status IS NULL OR status = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.db.Query(ctx, qRows, userID, f.Status, f.PerPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var listings []*domain.Listing
	for rows.Next() {
		var l domain.Listing
		if err := rows.Scan(
			&l.ID, &l.UserID, &l.Title, &l.Description, &l.Price,
			&l.Category, &l.Location, &l.Status, &l.IsPromoted,
			&l.PromotedUntil, &l.ViewsCount, &l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		listings = append(listings, &l)
	}
	return listings, total, rows.Err()
}
