package analytics

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB *pgxpool.Pool
}

// Summary is a coarse, ungrouped snapshot of platform metrics — carried
// over as-is from the old admin.Analytics type. It doesn't yet satisfy the
// assignment's "group by date/offer, impressions/clicks/DAU" requirement —
// that's a known gap tracked in the README for the next pass — but living
// under analytics now means that work extends this file instead of
// requiring another cross-package split.
type Summary struct {
	TotalUsers   int     `json:"total_users"`
	TotalRevenue float64 `json:"total_revenue"`
	ActiveOffers int     `json:"active_offers"`
}

// Create inserts one tracking event (impression | click) fired by the
// frontend. offer_id/user_id are nullable at the DB level (ON DELETE SET
// NULL), so this never fails due to a since-deleted offer or user.
func (r *Repository) Create(ctx context.Context, userID, offerID, eventType string) error {
	_, err := r.DB.Exec(ctx, `
		INSERT INTO events (type, offer_id, user_id)
		VALUES ($1, $2, $3)
	`, eventType, offerID, userID)
	return err
}

// GetSummary aggregates platform-wide totals in a single round trip.
func (r *Repository) GetSummary(ctx context.Context) (*Summary, error) {
	var s Summary
	err := r.DB.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM users) AS total_users,
			(SELECT COALESCE(SUM(amount), 0) FROM wallet_transactions WHERE amount > 0) AS total_revenue,
			(SELECT COUNT(*) FROM offers WHERE is_active = true) AS active_offers
	`).Scan(&s.TotalUsers, &s.TotalRevenue, &s.ActiveOffers)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
