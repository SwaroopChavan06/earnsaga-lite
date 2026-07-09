package leaderboard

import (
	"context"
)

type Service struct {
	Repo *Repository
}

type Entry struct {
	Name          string  `json:"name"`
	AvatarURL     string  `json:"avatar_url"`
	TotalEarnings float64 `json:"total_earnings"`
}

func (s *Service) GetLeaderboard(ctx context.Context) ([]Entry, error) {
	rows, err := s.Repo.GetTopUsers(ctx)
	if err != nil {
		return nil, err
	}

	var entries []Entry
	for _, r := range rows {
		entries = append(entries, Entry{
			Name:          r.Name,
			AvatarURL:     r.AvatarURL,
			TotalEarnings: r.TotalEarnings,
		})
	}
	return entries, nil
}
