package user

import (
	"context"
	"earnsaga-lite/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB *pgxpool.Pool
}

func (r *Repository) GetByID(ctx context.Context, userID string) (*models.User, error) {
	var u models.User
	err := r.DB.QueryRow(ctx, `
		SELECT id, email, name, avatar_url, is_admin FROM users WHERE id = $1
	`, userID).Scan(&u.ID, &u.Email, &u.Name, &u.AvatarURL, &u.IsAdmin)
	return &u, err
}
