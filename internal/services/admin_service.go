package services

import (
	"context"
	"earnsaga-lite/internal/repositories"
)

type AdminService struct {
	Repo *repositories.AdminRepository
}

func (s *AdminService) GetAnalytics(ctx context.Context) (*repositories.Analytics, error) {
	return s.Repo.GetAnalytics(ctx)
}
