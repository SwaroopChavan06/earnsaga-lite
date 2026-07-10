package leaderboard

import (
	"context"
	"testing"
)

type fakeRepo struct {
	scores          []ScoreRow
	info            map[string]UserInfo
	lastRange       string
	incrementCalls  int
	lastIncrementBy float64
}

func (f *fakeRepo) GetTopScores(ctx context.Context, rangeName string, limit int64) ([]ScoreRow, error) {
	f.lastRange = rangeName
	return f.scores, nil
}
func (f *fakeRepo) GetUserInfo(ctx context.Context, userIDs []string) (map[string]UserInfo, error) {
	return f.info, nil
}
func (f *fakeRepo) IncrementScore(ctx context.Context, userID string, amount float64) error {
	f.incrementCalls++
	f.lastIncrementBy = amount
	return nil
}

func TestGetLeaderboard_RanksInOrderAndEnriches(t *testing.T) {
	repo := &fakeRepo{
		scores: []ScoreRow{
			{UserID: "u1", Score: 100},
			{UserID: "u2", Score: 50},
			{UserID: "u3", Score: 10}, // no matching info entry below
		},
		info: map[string]UserInfo{
			"u1": {Name: "Ada", AvatarURL: "http://x/ada.png"},
			"u2": {Name: "Bob", AvatarURL: "http://x/bob.png"},
		},
	}
	svc := &Service{Repo: repo}

	entries, err := svc.GetLeaderboard(context.Background(), RangeWeekly)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	for i, e := range entries {
		if e.Rank != i+1 {
			t.Fatalf("expected rank %d at position %d, got %d", i+1, i, e.Rank)
		}
	}
	if entries[0].Name != "Ada" || entries[1].Name != "Bob" {
		t.Fatalf("expected names enriched from GetUserInfo, got %+v", entries)
	}
	if entries[2].Name != "" {
		t.Fatalf("expected a missing user-info entry to enrich with a zero value, not error, got %+v", entries[2])
	}
	if repo.lastRange != RangeWeekly {
		t.Fatalf("expected the requested range to be forwarded, got %q", repo.lastRange)
	}
}

func TestGetLeaderboard_InvalidRangeDefaultsToAllTime(t *testing.T) {
	repo := &fakeRepo{}
	svc := &Service{Repo: repo}

	if _, err := svc.GetLeaderboard(context.Background(), "not-a-real-range"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastRange != RangeAllTime {
		t.Fatalf("expected an unrecognized range to fall back to alltime, got %q", repo.lastRange)
	}
}

func TestRecordEarning_PassesThrough(t *testing.T) {
	repo := &fakeRepo{}
	svc := &Service{Repo: repo}

	if err := svc.RecordEarning(context.Background(), "u1", 12.5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.incrementCalls != 1 || repo.lastIncrementBy != 12.5 {
		t.Fatalf("expected the earning amount to be forwarded, got calls=%d amount=%v", repo.incrementCalls, repo.lastIncrementBy)
	}
}
