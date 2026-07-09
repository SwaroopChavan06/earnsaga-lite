package admin

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB *pgxpool.Pool
}

type Analytics struct {
	TotalUsers   int     `json:"total_users"`
	TotalRevenue float64 `json:"total_revenue"`
	ActiveOffers int     `json:"active_offers"`
}

func (r *Repository) GetAnalytics(ctx context.Context) (*Analytics, error) {
	var a Analytics
	err := r.DB.QueryRow(ctx, `
		SELECT 
			(SELECT COUNT(*) FROM users) as total_users,
			(SELECT COALESCE(SUM(amount), 0) FROM wallet_transactions WHERE amount > 0) as total_revenue,
			(SELECT COUNT(*) FROM offers WHERE is_active = true) as active_offers
	`).Scan(&a.TotalUsers, &a.TotalRevenue, &a.ActiveOffers)
	if err != nil {
		return nil, err
	}
	return &a, nil
}
