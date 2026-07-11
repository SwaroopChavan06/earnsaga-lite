package offer

import (
	"context"
	"errors"
	"strings"
	"testing"

	"earnsaga-lite/internal/models"
	"earnsaga-lite/internal/pubscale"

	"github.com/jackc/pgx/v5"
)

// fakeRepo is an in-memory stand-in for offer.Repository.
type fakeRepo struct {
	offer          *models.Offer
	goals          []models.OfferGoal
	status         string
	statusErr      error
	insertStartErr error
	insertCalls    int
}

func (f *fakeRepo) UpsertFromPubScale(ctx context.Context, o pubscale.Offer) error { return nil }
func (f *fakeRepo) ListActive(ctx context.Context, search string, limit, offset int) ([]models.Offer, int, error) {
	return nil, 0, nil
}
func (f *fakeRepo) GetByID(ctx context.Context, id string) (*models.Offer, error) {
	return f.offer, nil
}
func (f *fakeRepo) ListGoals(ctx context.Context, offerID string) ([]models.OfferGoal, error) {
	return f.goals, nil
}
func (f *fakeRepo) GetUserOfferStatus(ctx context.Context, userID, offerID string) (string, error) {
	return f.status, f.statusErr
}
func (f *fakeRepo) InsertStart(ctx context.Context, userID, offerID string) error {
	f.insertCalls++
	return f.insertStartErr
}

func newNotStartedRepo() *fakeRepo {
	return &fakeRepo{
		offer:     &models.Offer{ID: "offer-1", TrackingURL: "https://advertiser.example/click?uid={your_user_id}"},
		statusErr: pgx.ErrNoRows,
	}
}

func TestStart_FirstClick_InsertsAndSubstitutesUserID(t *testing.T) {
	repo := newNotStartedRepo()
	svc := &Service{Repo: repo}

	result, err := svc.Start(context.Background(), "offer-1", "user-42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AlreadyStarted {
		t.Fatal("expected AlreadyStarted=false on a first click")
	}
	if result.Status != string(models.StatusInProgress) {
		t.Fatalf("expected status in_progress, got %q", result.Status)
	}
	if !strings.Contains(result.RedirectURL, "uid=user-42") {
		t.Fatalf("expected {your_user_id} substituted with the real user id, got %q", result.RedirectURL)
	}
	if repo.insertCalls != 1 {
		t.Fatalf("expected exactly one InsertStart call, got %d", repo.insertCalls)
	}
}

func TestStart_RepeatClick_ReturnsCurrentStateWithoutErrorOrDuplicateInsert(t *testing.T) {
	repo := &fakeRepo{
		offer:  &models.Offer{ID: "offer-1", TrackingURL: "https://advertiser.example/click?uid={your_user_id}"},
		status: string(models.StatusInProgress),
	}
	svc := &Service{Repo: repo}

	result, err := svc.Start(context.Background(), "offer-1", "user-42")
	if err != nil {
		t.Fatalf("a repeat start must never error, got: %v", err)
	}
	if !result.AlreadyStarted {
		t.Fatal("expected AlreadyStarted=true on a repeat click")
	}
	if result.Status != string(models.StatusInProgress) {
		t.Fatalf("expected the existing status to be echoed back, got %q", result.Status)
	}
	if repo.insertCalls != 0 {
		t.Fatalf("a repeat click must not insert a second user_offers row, got %d inserts", repo.insertCalls)
	}
}

func TestStart_RepeatClickAfterCompletion_ReturnsCompletedState(t *testing.T) {
	repo := &fakeRepo{
		offer:  &models.Offer{ID: "offer-1", TrackingURL: "https://advertiser.example/click?uid={your_user_id}"},
		status: string(models.StatusCompleted),
	}
	svc := &Service{Repo: repo}

	result, err := svc.Start(context.Background(), "offer-1", "user-42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != string(models.StatusCompleted) {
		t.Fatalf("expected completed status to be preserved, got %q", result.Status)
	}
}

func TestStart_RealDBErrorPropagates(t *testing.T) {
	repo := &fakeRepo{
		offer:     &models.Offer{ID: "offer-1"},
		statusErr: errors.New("connection reset"),
	}
	svc := &Service{Repo: repo}

	_, err := svc.Start(context.Background(), "offer-1", "user-42")
	if err == nil {
		t.Fatal("expected a real DB error (not pgx.ErrNoRows) to propagate, not be swallowed")
	}
}

func TestGetDetail_NotStartedWhenNoUserOrNoRow(t *testing.T) {
	repo := newNotStartedRepo()
	repo.goals = []models.OfferGoal{{ID: "goal-1", Title: "Reach level 5"}}
	svc := &Service{Repo: repo}

	detail, err := svc.GetDetail(context.Background(), "offer-1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Status != string(models.StatusNotStarted) {
		t.Fatalf("expected not_started with no user id supplied, got %q", detail.Status)
	}
	if len(detail.Goals) != 1 {
		t.Fatalf("expected goals to be included in the detail response, got %d", len(detail.Goals))
	}

	detail, err = svc.GetDetail(context.Background(), "offer-1", "user-42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Status != string(models.StatusNotStarted) {
		t.Fatalf("expected not_started when the user has no user_offers row, got %q", detail.Status)
	}
}

func TestGetDetail_ReflectsInProgressStatus(t *testing.T) {
	repo := &fakeRepo{
		offer:  &models.Offer{ID: "offer-1"},
		status: string(models.StatusInProgress),
	}
	svc := &Service{Repo: repo}

	detail, err := svc.GetDetail(context.Background(), "offer-1", "user-42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Status != string(models.StatusInProgress) {
		t.Fatalf("expected in_progress, got %q", detail.Status)
	}
}
