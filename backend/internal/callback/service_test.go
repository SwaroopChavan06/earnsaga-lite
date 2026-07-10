package callback

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"

	"earnsaga-lite/internal/leaderboard"
)

func validSignature(secret, userID string, value float64, token string) string {
	sum := md5.Sum([]byte(fmt.Sprintf("%s.%s.%d.%s", secret, userID, int64(value), token)))
	return hex.EncodeToString(sum[:])
}

func TestVerifySignature(t *testing.T) {
	s := &Service{SecretKey: "top-secret"}
	userID, value, token := "user-123", 50.0, "tok-abc"
	sig := validSignature(s.SecretKey, userID, value, token)

	if !s.VerifySignature(userID, value, token, sig) {
		t.Fatal("expected a correctly computed signature to verify")
	}
	if s.VerifySignature(userID, value, token, sig[:len(sig)-1]+"0") {
		t.Fatal("expected a tampered (single character changed) signature to fail")
	}
	if s.VerifySignature(userID, value, token, "") {
		t.Fatal("expected an empty signature to fail")
	}
	wrongSecret := &Service{SecretKey: "different-secret"}
	if wrongSecret.VerifySignature(userID, value, token, sig) {
		t.Fatal("expected a signature computed with a different secret to fail")
	}
	// Changing any input the signature is derived from must invalidate it.
	if s.VerifySignature(userID, value+1, token, sig) {
		t.Fatal("expected signature to fail when value differs from what it was computed for")
	}
}

// fakeRepo is an in-memory stand-in for callback.Repository, keyed on
// token so idempotency behavior matches the real unique-constraint-backed
// implementation without needing a database.
type fakeRepo struct {
	credited map[string]bool
	callErr  error
	calls    int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{credited: map[string]bool{}}
}

func (f *fakeRepo) CreditWalletIdempotent(ctx context.Context, userID string, value float64, token string) (bool, error) {
	f.calls++
	if f.callErr != nil {
		return false, f.callErr
	}
	if f.credited[token] {
		return false, nil
	}
	f.credited[token] = true
	return true, nil
}

// fakeLeaderboardRepo satisfies leaderboard's own unexported repository
// interface structurally (see internal/leaderboard/service.go) — its
// methods only need to match by signature, not by importing that
// interface type, which is why this works across package boundaries.
type fakeLeaderboardRepo struct {
	incrementCalls int
	incrementErr   error
}

func (f *fakeLeaderboardRepo) GetTopScores(ctx context.Context, rangeName string, limit int64) ([]leaderboard.ScoreRow, error) {
	return nil, nil
}
func (f *fakeLeaderboardRepo) GetUserInfo(ctx context.Context, userIDs []string) (map[string]leaderboard.UserInfo, error) {
	return nil, nil
}
func (f *fakeLeaderboardRepo) IncrementScore(ctx context.Context, userID string, amount float64) error {
	f.incrementCalls++
	return f.incrementErr
}

func TestHandleCallback_CreditsOnce(t *testing.T) {
	repo := newFakeRepo()
	lbRepo := &fakeLeaderboardRepo{}
	svc := &Service{Repo: repo, Leaderboard: &leaderboard.Service{Repo: lbRepo}}

	credited, err := svc.HandleCallback(context.Background(), "user-1", 25.0, "tok-1")
	if err != nil || !credited {
		t.Fatalf("first callback: want credited=true err=nil, got credited=%v err=%v", credited, err)
	}
	if lbRepo.incrementCalls != 1 {
		t.Fatalf("expected leaderboard to be updated once, got %d calls", lbRepo.incrementCalls)
	}

	// Replay of the same token must not credit again, and must not touch
	// the leaderboard a second time either — a replay must not
	// double-count a wallet credit OR a ranking.
	credited, err = svc.HandleCallback(context.Background(), "user-1", 25.0, "tok-1")
	if err != nil || credited {
		t.Fatalf("replayed callback: want credited=false err=nil, got credited=%v err=%v", credited, err)
	}
	if lbRepo.incrementCalls != 1 {
		t.Fatalf("expected leaderboard NOT to be updated again on replay, got %d total calls", lbRepo.incrementCalls)
	}
	if repo.calls != 2 {
		t.Fatalf("expected exactly 2 repository calls (one per HandleCallback invocation), got %d", repo.calls)
	}
}

func TestHandleCallback_LeaderboardFailureDoesNotFailCallback(t *testing.T) {
	repo := newFakeRepo()
	lbRepo := &fakeLeaderboardRepo{incrementErr: errors.New("redis unavailable")}
	svc := &Service{Repo: repo, Leaderboard: &leaderboard.Service{Repo: lbRepo}}

	credited, err := svc.HandleCallback(context.Background(), "user-1", 10.0, "tok-2")
	if err != nil || !credited {
		t.Fatalf("a leaderboard failure must not fail an otherwise-successful wallet credit, got credited=%v err=%v", credited, err)
	}
}

func TestHandleCallback_WithoutLeaderboardWired(t *testing.T) {
	repo := newFakeRepo()
	svc := &Service{Repo: repo} // Leaderboard left nil on purpose

	credited, err := svc.HandleCallback(context.Background(), "user-1", 10.0, "tok-3")
	if err != nil || !credited {
		t.Fatalf("callback must still succeed with no leaderboard wired, got credited=%v err=%v", credited, err)
	}
}

func TestHandleCallback_RepositoryErrorPropagates(t *testing.T) {
	repo := newFakeRepo()
	repo.callErr = errors.New("db down")
	svc := &Service{Repo: repo}

	credited, err := svc.HandleCallback(context.Background(), "user-1", 10.0, "tok-4")
	if err == nil {
		t.Fatal("expected repository error to propagate")
	}
	if credited {
		t.Fatal("expected credited=false on error")
	}
}
