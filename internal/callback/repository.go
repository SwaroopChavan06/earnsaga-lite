package callback

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB *pgxpool.Pool
}

// CreditWalletIdempotent inserts a wallet_transactions row keyed by the
// callback token. The token column has a unique constraint, so a replayed
// callback (same token) is a no-op insert — credited comes back false and
// nothing is double-counted. This is what makes it safe for PubScale to
// retry a callback we already processed.
func (r *Repository) CreditWalletIdempotent(ctx context.Context, userID string, value float64, token string) (credited bool, err error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	var txID string
	err = tx.QueryRow(ctx, `
		INSERT INTO wallet_transactions (user_id, amount, type, callback_token)
		VALUES ($1, $2, 'credit', $3)
		ON CONFLICT (callback_token) DO NOTHING
		RETURNING id
	`, userID, value, token).Scan(&txID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Already processed — not a real error, just nothing new to do.
			return false, tx.Commit(ctx)
		}
		return false, err
	}

	// Best-effort completion marking: PubScale's callback payload only
	// gives us user_id/value/token/signature, not which offer/goal this
	// credit is for. Marking the oldest in-progress offer as completed is
	// a reasonable heuristic for this build. Stricter attribution would
	// need the offer_id encoded as a custom param in the tracking URL
	// that PubScale echoes back in the callback.
	_, err = tx.Exec(ctx, `
		UPDATE user_offers SET status = 'completed', completed_at = now()
		WHERE id = (
			SELECT id FROM user_offers
			WHERE user_id = $1 AND status = 'in_progress'
			ORDER BY started_at ASC LIMIT 1
		)
	`, userID)
	if err != nil {
		return false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}
