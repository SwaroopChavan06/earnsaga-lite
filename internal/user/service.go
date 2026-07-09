package user

import (
	"context"
)

type UserProfileResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	IsAdmin   bool   `json:"is_admin"`
}

type Service struct {
	Repo *Repository
}

func (s *Service) GetProfile(ctx context.Context, userID string) (*UserProfileResponse, error) {
	user, err := s.Repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &UserProfileResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		AvatarURL: user.AvatarURL,
		IsAdmin:   user.IsAdmin,
	}, nil
}
