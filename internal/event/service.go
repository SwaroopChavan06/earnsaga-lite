package event

import (
	"context"
)

type Service struct {
	Repo *Repository
}

func (s *Service) Track(ctx context.Context, userID, offerID, eventType string) error {
	return s.Repo.Create(ctx, userID, offerID, eventType)
}
