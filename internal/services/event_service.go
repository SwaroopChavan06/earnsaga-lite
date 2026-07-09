package services

import (
	"context"
	"earnsaga-lite/internal/repositories"
)

type EventService struct {
	Repo *repositories.EventRepository
}

func (s *EventService) Track(ctx context.Context, userID, offerID, eventType string) error {
	return s.Repo.Create(ctx, userID, offerID, eventType)
}
