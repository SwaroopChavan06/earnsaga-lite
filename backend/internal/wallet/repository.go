package wallet

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB *pgxpool.Pool
}

// TransactionRow is the raw shape read back from wallet_transactions,
// joined to offers for a display name. offer_id/goal_id/offer_name are
// nullable — a transaction may be unattributed if the callback that
// created it couldn't be matched to any in-progress offer.
type TransactionRow struct {
	ID        string
	Amount    float64
	Type      string
	OfferID   *string
	GoalID    *string
	OfferName *string
	CreatedAt time.Time
}

func (r *Repository) GetBalance(ctx context.Context, userID string) (float64, error) {
	var balance float64
	err := r.DB.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM wallet_transactions WHERE user_id = $1
	`, userID).Scan(&balance)
	return balance, err
}

func (r *Repository) GetTransactions(ctx context.Context, userID string) ([]TransactionRow, error) {
	rows, err := r.DB.Query(ctx, `
		SELECT wt.id, wt.amount, wt.type, wt.offer_id, wt.goal_id, o.name, wt.created_at
		FROM wallet_transactions wt
		LEFT JOIN offers o ON o.id = wt.offer_id
		WHERE wt.user_id = $1
		ORDER BY wt.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	txs := []TransactionRow{}
	for rows.Next() {
		var t TransactionRow
		if err := rows.Scan(&t.ID, &t.Amount, &t.Type, &t.OfferID, &t.GoalID, &t.OfferName, &t.CreatedAt); err != nil {
			return nil, err
		}
		txs = append(txs, t)
	}
	return txs, rows.Err()
}
