package callback

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB *pgxpool.Pool
}

// attributionMatch is the in-progress offer/goal (if any) a callback's
// credit should be attributed to. Pointer fields are nil when there's
// nothing to attribute to (user has no in-progress offer at all) or when
// a fallback match couldn't identify a specific goal.
type attributionMatch struct {
	userOfferID *string
	offerID     *string
	goalID      *string
}

// offerCandidate is one (in-progress offer, goal) row a callback could be
// attributed to. Multiple candidates share the same userOfferID/offerID
// when an offer has more than one goal.
type offerCandidate struct {
	userOfferID string
	offerID     string
	startedAt   time.Time
	goalID      *string
	goalReward  *float64
}

// selectAttributionMatch is a pure function (no DB access) that picks the
// best in-progress offer/goal for a callback of the given value.
//
// PubScale's S2S callback (per the assignment doc) only sends
// user_id/value/token/signature -- there's no offer or goal id to key off
// directly. This is the best available signal: prefer an in-progress goal
// whose configured reward exactly matches the callback value, and only
// fall back to "oldest in-progress offer" when no goal's reward matches
// anything. It's still a heuristic, just one grounded in real data instead
// of a blind guess.
//
// candidates must be pre-sorted by startedAt ascending (oldest first) --
// the SQL query that produces them already orders this way, so ties on
// the reward match resolve to the earliest-started offer. Kept as its own
// function (rather than inlined in the DB call) specifically so it's
// unit-testable without a database.
func selectAttributionMatch(candidates []offerCandidate, value float64) attributionMatch {
	if len(candidates) == 0 {
		return attributionMatch{}
	}

	target := roundToCents(value)
	for _, c := range candidates {
		if c.goalReward != nil && roundToCents(*c.goalReward) == target {
			userOfferID, offerID := c.userOfferID, c.offerID
			return attributionMatch{userOfferID: &userOfferID, offerID: &offerID, goalID: c.goalID}
		}
	}

	// No reward matched anywhere -- fall back to the oldest in-progress
	// offer overall. candidates[0] is guaranteed to be it since the query
	// orders by started_at ascending.
	first := candidates[0]
	userOfferID, offerID := first.userOfferID, first.offerID
	return attributionMatch{userOfferID: &userOfferID, offerID: &offerID}
}

func roundToCents(v float64) float64 {
	return math.Round(v*100) / 100
}

// findAttributionCandidates loads every (in-progress offer, goal) pair for
// the user, oldest offer first, for selectAttributionMatch to choose from.
func findAttributionCandidates(ctx context.Context, tx pgx.Tx, userID string) ([]offerCandidate, error) {
	rows, err := tx.Query(ctx, `
		SELECT uo.id, uo.offer_id, uo.started_at, og.id, og.reward
		FROM user_offers uo
		LEFT JOIN offer_goals og ON og.offer_id = uo.offer_id
		WHERE uo.user_id = $1 AND uo.status = 'in_progress'
		ORDER BY uo.started_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []offerCandidate
	for rows.Next() {
		var c offerCandidate
		if err := rows.Scan(&c.userOfferID, &c.offerID, &c.startedAt, &c.goalID, &c.goalReward); err != nil {
			return nil, err
		}
		candidates = append(candidates, c)
	}
	return candidates, rows.Err()
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

	candidates, err := findAttributionCandidates(ctx, tx, userID)
	if err != nil {
		return false, err
	}
	match := selectAttributionMatch(candidates, value)

	var txID string
	err = tx.QueryRow(ctx, `
		INSERT INTO wallet_transactions (user_id, amount, offer_id, goal_id, type, callback_token)
		VALUES ($1, $2, $3, $4, 'credit', $5)
		ON CONFLICT (callback_token) DO NOTHING
		RETURNING id
	`, userID, value, match.offerID, match.goalID, token).Scan(&txID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Already processed — not a real error, just nothing new to do.
			return false, tx.Commit(ctx)
		}
		return false, err
	}

	if match.userOfferID != nil {
		if _, err := tx.Exec(ctx, `
			UPDATE user_offers SET status = 'completed', completed_at = now() WHERE id = $1
		`, *match.userOfferID); err != nil {
			return false, err
		}

		if match.goalID != nil {
			if _, err := tx.Exec(ctx, `
				INSERT INTO user_offer_goals (user_offer_id, goal_id, completed_at)
				VALUES ($1, $2, now())
				ON CONFLICT (user_offer_id, goal_id) DO UPDATE SET completed_at = now()
			`, *match.userOfferID, *match.goalID); err != nil {
				return false, err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}
