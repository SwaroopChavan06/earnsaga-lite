package user

import (
	"context"

	"earnsaga-lite/internal/models"
)

type UserProfileResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	IsAdmin   bool   `json:"is_admin"`
}

// repository is the seam user.Service depends on. Unexported since it only
// exists to let tests substitute a fake — the concrete *Repository (in
// repository.go) satisfies it implicitly, so main.go wiring is untouched.
type repository interface {
	GetByID(ctx context.Context, userID string) (*models.User, error)
	FindOrCreateByGoogle(ctx context.Context, sub, email, name, avatarURL string) (*models.User, error)
	IsAdmin(ctx context.Context, userID string) (bool, error)
}

type Service struct {
	Repo repository
}

func (s *Service) GetProfile(ctx context.Context, userID string) (*UserProfileResponse, error) {
	user, err := s.Repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return toProfile(user), nil
}

// FindOrCreateByGoogle is called by auth.Handler right after a Google
// id_token (or, in dev mode, a synthetic sub) has been verified — user
// creation/lookup is this domain's responsibility, not auth's. Returns
// *models.User (not the UserProfileResponse DTO) so auth.Handler can depend
// on a locally-defined interface over the shared models package instead of
// importing this package directly, which would create an auth<->user
// import cycle (user.Handler already imports auth for context extraction).
func (s *Service) FindOrCreateByGoogle(ctx context.Context, sub, email, name, avatarURL string) (*models.User, error) {
	return s.Repo.FindOrCreateByGoogle(ctx, sub, email, name, avatarURL)
}

// IsAdmin is used by auth.RequireAdmin to gate admin-only routes.
func (s *Service) IsAdmin(ctx context.Context, userID string) (bool, error) {
	return s.Repo.IsAdmin(ctx, userID)
}

func toProfile(u *models.User) *UserProfileResponse {
	return &UserProfileResponse{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		AvatarURL: u.AvatarURL,
		IsAdmin:   u.IsAdmin,
	}
}
