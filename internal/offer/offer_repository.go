package offer

import (
	"context"

	"earnsaga-lite/internal/models"
	"earnsaga-lite/internal/pubscale"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB *pgxpool.Pool
}

// UpsertFromPubScale writes one PubScale offer + its goals in a single
// transaction. Safe to call repeatedly — pubscale_id and (offer_id,
// pubscale_goal_id) both have unique constraints, so re-syncing updates
// existing rows instead of duplicating.
func (r *Repository) UpsertFromPubScale(ctx context.Context, o pubscale.Offer) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // no-op if committed

	var offerID string
	err = tx.QueryRow(ctx, `
		INSERT INTO offers (pubscale_id, name, icon_url, description, total_payout, tracking_url, is_active, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, TRUE, now())
		ON CONFLICT (pubscale_id) DO UPDATE SET
			name = EXCLUDED.name,
			icon_url = EXCLUDED.icon_url,
			description = EXCLUDED.description,
			total_payout = EXCLUDED.total_payout,
			tracking_url = EXCLUDED.tracking_url,
			is_active = TRUE,
			updated_at = now()
		RETURNING id
	`, o.ID, o.Name, o.Creative.IconURL, o.Desc.Raw, o.Payout.Amount, o.TrackURL).Scan(&offerID)
	if err != nil {
		return err
	}

	for _, g := range o.Goals {
		_, err = tx.Exec(ctx, `
			INSERT INTO offer_goals (offer_id, pubscale_goal_id, title, instructions, reward, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (offer_id, pubscale_goal_id) DO UPDATE SET
				title = EXCLUDED.title,
				instructions = EXCLUDED.instructions,
				reward = EXCLUDED.reward,
				sort_order = EXCLUDED.sort_order
		`, offerID, g.ID, g.Title, g.Instructions, g.Payout.Amount, g.Order)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) ListActive(ctx context.Context, search string) ([]models.Offer, error) {
	var rows pgx.Rows
	var err error
	if search == "" {
		rows, err = r.DB.Query(ctx, `
			SELECT id, name, icon_url, total_payout FROM offers
			WHERE is_active = TRUE ORDER BY created_at DESC LIMIT 100
		`)
	} else {
		rows, err = r.DB.Query(ctx, `
			SELECT id, name, icon_url, total_payout FROM offers
			WHERE is_active = TRUE AND name ILIKE '%' || $1 || '%'
			ORDER BY created_at DESC LIMIT 100
		`, search)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	offers := []models.Offer{}
	for rows.Next() {
		var o models.Offer
		if err := rows.Scan(&o.ID, &o.Name, &o.IconURL, &o.TotalPayout); err != nil {
			return nil, err
		}
		offers = append(offers, o)
	}
	return offers, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id string) (*models.Offer, error) {
	var o models.Offer
	err := r.DB.QueryRow(ctx, `
		SELECT id, name, icon_url, description, total_payout, tracking_url FROM offers
		WHERE id = $1 AND is_active = TRUE
	`, id).Scan(&o.ID, &o.Name, &o.IconURL, &o.Description, &o.TotalPayout, &o.TrackingURL)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *Repository) ListGoals(ctx context.Context, offerID string) ([]models.OfferGoal, error) {
	rows, err := r.DB.Query(ctx, `
		SELECT id, title, instructions, reward FROM offer_goals
		WHERE offer_id = $1 ORDER BY sort_order ASC
	`, offerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	goals := []models.OfferGoal{}
	for rows.Next() {
		var g models.OfferGoal
		if err := rows.Scan(&g.ID, &g.Title, &g.Instructions, &g.Reward); err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	return goals, rows.Err()
}

// GetUserOfferStatus returns pgx.ErrNoRows if the user hasn't started this
// offer — callers should treat that as "not_started", not a real error.
func (r *Repository) GetUserOfferStatus(ctx context.Context, userID, offerID string) (string, error) {
	var status string
	err := r.DB.QueryRow(ctx, `
		SELECT status FROM user_offers WHERE user_id = $1 AND offer_id = $2
	`, userID, offerID).Scan(&status)
	return status, err
}

// InsertStart creates the user_offers row for a fresh start. Relies on the
// (user_id, offer_id) unique constraint — ON CONFLICT DO NOTHING makes this
// safe to call even in a rare race between two "check then insert" calls.
func (r *Repository) InsertStart(ctx context.Context, userID, offerID string) error {
	_, err := r.DB.Exec(ctx, `
		INSERT INTO user_offers (user_id, offer_id, status)
		VALUES ($1, $2, 'in_progress')
		ON CONFLICT (user_id, offer_id) DO NOTHING
	`, userID, offerID)
	return err
}
