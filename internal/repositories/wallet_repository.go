package repositories

import (
	"context"
	"earnsaga-lite/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WalletRepository struct {
	DB *pgxpool.Pool
}

func (r *WalletRepository) GetBalance(ctx context.Context, userID string) (float64, error) {
	var balance float64
	err := r.DB.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM wallet_transactions WHERE user_id = $1
	`, userID).Scan(&balance)
	return balance, err
}

func (r *WalletRepository) GetTransactions(ctx context.Context, userID string) ([]models.WalletTransaction, error) {
	rows, err := r.DB.Query(ctx, `
		SELECT id, amount, type, created_at FROM wallet_transactions WHERE user_id = $1 ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []models.WalletTransaction
	for rows.Next() {
		var t models.WalletTransaction
		if err := rows.Scan(&t.ID, &t.Amount, &t.Type, &t.CreatedAt); err != nil {
			return nil, err
		}
		txs = append(txs, t)
	}
	return txs, nil
}
