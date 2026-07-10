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
		SELECT id, email, name, avatar_url, is_admin, created_at FROM users WHERE id = $1
	`, userID).Scan(&u.ID, &u.Email, &u.Name, &u.AvatarURL, &u.IsAdmin, &u.CreatedAt)
	return &u, err
}

// FindOrCreateByGoogle upserts on google_sub — first login creates the row,
// every later login just refreshes the profile fields Google may have
// updated (name/avatar change, email stays the natural key alongside sub).
func (r *Repository) FindOrCreateByGoogle(ctx context.Context, sub, email, name, avatarURL string) (*models.User, error) {
	var u models.User
	err := r.DB.QueryRow(ctx, `
		INSERT INTO users (google_sub, email, name, avatar_url)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (google_sub) DO UPDATE SET
			email = EXCLUDED.email,
			name = EXCLUDED.name,
			avatar_url = EXCLUDED.avatar_url
		RETURNING id, email, name, avatar_url, is_admin, created_at
	`, sub, email, name, avatarURL).Scan(&u.ID, &u.Email, &u.Name, &u.AvatarURL, &u.IsAdmin, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// IsAdmin is used by auth.RequireAdmin to gate admin-only routes.
func (r *Repository) IsAdmin(ctx context.Context, userID string) (bool, error) {
	var isAdmin bool
	err := r.DB.QueryRow(ctx, `SELECT is_admin FROM users WHERE id = $1`, userID).Scan(&isAdmin)
	return isAdmin, err
}
