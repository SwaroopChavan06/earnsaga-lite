package services

import (
	"context"
	"earnsaga-lite/internal/repositories"
)

type LeaderboardService struct {
	Repo *repositories.LeaderboardRepository
}

type LeaderboardEntry struct {
	Name          string  `json:"name"`
	AvatarURL     string  `json:"avatar_url"`
	TotalEarnings float64 `json:"total_earnings"`
}

func (s *LeaderboardService) GetLeaderboard(ctx context.Context) ([]LeaderboardEntry, error) {
	rows, err := s.Repo.GetTopUsers(ctx)
	if err != nil {
		return nil, err
	}

	var entries []LeaderboardEntry
	for _, r := range rows {
		entries = append(entries, LeaderboardEntry{
			Name:          r.Name,
			AvatarURL:     r.AvatarURL,
			TotalEarnings: r.TotalEarnings,
		})
	}
	return entries, nil
}
