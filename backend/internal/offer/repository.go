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
//
// category/platform/offer_type carry data PubScale already sends (ctg,
// os, off_type) that was previously parsed by the client but discarded —
// they're persisted here so the offer detail page can show them.
func (r *Repository) UpsertFromPubScale(ctx context.Context, o pubscale.Offer) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // no-op if committed

	var offerID string
	err = tx.QueryRow(ctx, `
		INSERT INTO offers (pubscale_id, name, icon_url, description, total_payout, tracking_url, category, platform, offer_type, is_active, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, TRUE, now())
		ON CONFLICT (pubscale_id) DO UPDATE SET
			name = EXCLUDED.name,
			icon_url = EXCLUDED.icon_url,
			description = EXCLUDED.description,
			total_payout = EXCLUDED.total_payout,
			tracking_url = EXCLUDED.tracking_url,
			category = EXCLUDED.category,
			platform = EXCLUDED.platform,
			offer_type = EXCLUDED.offer_type,
			is_active = TRUE,
			updated_at = now()
		RETURNING id
	`, o.ID, o.Name, o.Creative.IconURL, o.Desc.Raw, o.Payout.Amount, o.TrackURL, o.Category, o.OS, o.OffType).Scan(&offerID)
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

// offerListColumns and offerDetailColumns are kept as named constants so
// both queries below (and their Scan calls) can't silently drift apart —
// pubscale_id and description were previously missing here entirely,
// which is why they always came back empty regardless of what was
// actually stored.
const offerListColumns = `id, pubscale_id, name, icon_url, description, total_payout, is_active, created_at, updated_at`
const offerDetailColumns = `id, pubscale_id, name, icon_url, description, total_payout, tracking_url, category, platform, offer_type, is_active, created_at, updated_at`

func scanOfferListRow(rows pgx.Rows, o *models.Offer, total *int) error {
	return rows.Scan(&o.ID, &o.PubScaleID, &o.Name, &o.IconURL, &o.Description, &o.TotalPayout, &o.IsActive, &o.CreatedAt, &o.UpdatedAt, total)
}

// ListActive returns one page of active offers plus the total matching count.
// COUNT(*) OVER() is a window function that returns the full count in the
// same query so we don't need a separate SELECT COUNT(*) round-trip.
func (r *Repository) ListActive(ctx context.Context, search string, limit, offset int) ([]models.Offer, int, error) {
	var rows pgx.Rows
	var err error
	if search == "" {
		rows, err = r.DB.Query(ctx, `
			SELECT `+offerListColumns+`, COUNT(*) OVER() AS total_count
			FROM offers
			WHERE is_active = TRUE
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2
		`, limit, offset)
	} else {
		rows, err = r.DB.Query(ctx, `
			SELECT `+offerListColumns+`, COUNT(*) OVER() AS total_count
			FROM offers
			WHERE is_active = TRUE AND name ILIKE '%' || $1 || '%'
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		`, search, limit, offset)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var total int
	offers := []models.Offer{}
	for rows.Next() {
		var o models.Offer
		if err := scanOfferListRow(rows, &o, &total); err != nil {
			return nil, 0, err
		}
		offers = append(offers, o)
	}
	return offers, total, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id string) (*models.Offer, error) {
	var o models.Offer
	err := r.DB.QueryRow(ctx, `
		SELECT `+offerDetailColumns+`
		FROM offers WHERE id = $1 AND is_active = TRUE
	`, id).Scan(&o.ID, &o.PubScaleID, &o.Name, &o.IconURL, &o.Description, &o.TotalPayout, &o.TrackingURL, &o.Category, &o.Platform, &o.OfferType, &o.IsActive, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *Repository) ListGoals(ctx context.Context, offerID string) ([]models.OfferGoal, error) {
	rows, err := r.DB.Query(ctx, `
		SELECT id, offer_id, title, instructions, reward, sort_order FROM offer_goals
		WHERE offer_id = $1 ORDER BY sort_order ASC
	`, offerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	goals := []models.OfferGoal{}
	for rows.Next() {
		var g models.OfferGoal
		if err := rows.Scan(&g.ID, &g.OfferID, &g.Title, &g.Instructions, &g.Reward, &g.SortOrder); err != nil {
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
