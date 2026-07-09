package services

import (
	"context"
	"earnsaga-lite/internal/dto"
	"earnsaga-lite/internal/repositories"
)

type UserService struct {
	Repo *repositories.UserRepository
}

func (s *UserService) GetProfile(ctx context.Context, userID string) (*dto.UserProfileResponse, error) {
	user, err := s.Repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &dto.UserProfileResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		AvatarURL: user.AvatarURL,
		IsAdmin:   user.IsAdmin,
	}, nil
}
