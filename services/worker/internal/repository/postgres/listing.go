package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ListingRepo struct {
	db *pgxpool.Pool
}

func NewListingRepo(db *pgxpool.Pool) *ListingRepo {
	return &ListingRepo{db: db}
}

func (r *ListingRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	const q = `UPDATE listings SET status = $1 WHERE id = $2`
	tag, err := r.db.Exec(ctx, q, status, id)
	if err != nil {
		return fmt.Errorf("update listing status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("listing %s not found", id)
	}
	return nil
}
