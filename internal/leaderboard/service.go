package leaderboard

import "context"

type Entry struct {
	Rank      int     `json:"rank"`
	UserID    string  `json:"user_id"`
	Name      string  `json:"name"`
	AvatarURL string  `json:"avatar_url"`
	Coins     float64 `json:"coins"`
}

type Service struct {
	Repo *Repository
}

// GetLeaderboard returns the top 50 for the given range, sourced entirely
// from Redis plus one bounded name/avatar lookup — never a full scan of
// wallet_transactions.
func (s *Service) GetLeaderboard(ctx context.Context, rangeName string) ([]Entry, error) {
	if rangeName != RangeDaily && rangeName != RangeWeekly {
		rangeName = RangeAllTime
	}

	scores, err := s.Repo.GetTopScores(ctx, rangeName, 50)
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(scores))
	for i, sc := range scores {
		ids[i] = sc.UserID
	}
	info, err := s.Repo.GetUserInfo(ctx, ids)
	if err != nil {
		return nil, err
	}

	entries := make([]Entry, len(scores))
	for i, sc := range scores {
		u := info[sc.UserID]
		entries[i] = Entry{
			Rank:      i + 1,
			UserID:    sc.UserID,
			Name:      u.Name,
			AvatarURL: u.AvatarURL,
			Coins:     sc.Score,
		}
	}
	return entries, nil
}

// RecordEarning is called by the callback domain right after a wallet
// credit succeeds — pushes the amount into daily/weekly/all-time sorted
// sets in one Redis pipeline.
func (s *Service) RecordEarning(ctx context.Context, userID string, amount float64) error {
	return s.Repo.IncrementScore(ctx, userID, amount)
}
