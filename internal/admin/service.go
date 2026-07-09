package admin

import (
	"context"
)

type Service struct {
	Repo *Repository
}

func (s *Service) GetAnalytics(ctx context.Context) (*Analytics, error) {
	return s.Repo.GetAnalytics(ctx)
}
