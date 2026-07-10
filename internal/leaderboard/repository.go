package leaderboard

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB *pgxpool.Pool
}

type Row struct {
	Name          string
	AvatarURL     string
	TotalEarnings float64
}

func (r *Repository) GetTopUsers(ctx context.Context) ([]Row, error) {
	rows, err := r.DB.Query(ctx, `
		SELECT u.name, u.avatar_url, SUM(wt.amount) as total_earnings
		FROM users u
		JOIN wallet_transactions wt ON u.id = wt.user_id
		GROUP BY u.id, u.name, u.avatar_url
		ORDER BY total_earnings DESC
		LIMIT 50
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []Row
	for rows.Next() {
		var row Row
		if err := rows.Scan(&row.Name, &row.AvatarURL, &row.TotalEarnings); err != nil {
			return nil, err
		}
		results = append(results, row)
	}
	return results, nil
}
