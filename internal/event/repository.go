package event

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB *pgxpool.Pool
}

func (r *Repository) Create(ctx context.Context, userID, offerID, eventType string) error {
	_, err := r.DB.Exec(ctx, `
		INSERT INTO events (type, offer_id, user_id)
		VALUES ($1, $2, $3)
	`, eventType, offerID, userID)
	return err
}
