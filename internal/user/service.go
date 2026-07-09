package user

import (
	"context"
)

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
