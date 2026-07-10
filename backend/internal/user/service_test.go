package user

import (
	"context"
	"errors"
	"testing"

	"earnsaga-lite/internal/models"
)

type fakeRepo struct {
	user      *models.User
	getErr    error
	isAdmin   bool
	adminErr  error
	upsertErr error
	lastSub   string
}

func (f *fakeRepo) GetByID(ctx context.Context, userID string) (*models.User, error) {
	return f.user, f.getErr
}
func (f *fakeRepo) FindOrCreateByGoogle(ctx context.Context, sub, email, name, avatarURL string) (*models.User, error) {
	f.lastSub = sub
	if f.upsertErr != nil {
		return nil, f.upsertErr
	}
	return f.user, nil
}
func (f *fakeRepo) IsAdmin(ctx context.Context, userID string) (bool, error) {
	return f.isAdmin, f.adminErr
}

func TestGetProfile_MapsFieldsCorrectly(t *testing.T) {
	repo := &fakeRepo{user: &models.User{
		ID: "u1", Email: "a@b.com", Name: "Ada", AvatarURL: "http://x/a.png", IsAdmin: true,
		GoogleSub: "should-not-leak-into-response",
	}}
	svc := &Service{Repo: repo}

	profile, err := svc.GetProfile(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile.ID != "u1" || profile.Email != "a@b.com" || profile.Name != "Ada" || !profile.IsAdmin {
		t.Fatalf("profile fields did not map correctly: %+v", profile)
	}
}

func TestGetProfile_PropagatesRepoError(t *testing.T) {
	repo := &fakeRepo{getErr: errors.New("not found")}
	svc := &Service{Repo: repo}

	if _, err := svc.GetProfile(context.Background(), "missing"); err == nil {
		t.Fatal("expected repository error to propagate")
	}
}

func TestFindOrCreateByGoogle_PassesSubThrough(t *testing.T) {
	repo := &fakeRepo{user: &models.User{ID: "u2", Email: "b@c.com"}}
	svc := &Service{Repo: repo}

	u, err := svc.FindOrCreateByGoogle(context.Background(), "google-sub-123", "b@c.com", "Bob", "http://x/b.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != "u2" {
		t.Fatalf("expected the repo's returned user to pass through unchanged, got %+v", u)
	}
	if repo.lastSub != "google-sub-123" {
		t.Fatalf("expected the google sub to be forwarded to the repository, got %q", repo.lastSub)
	}
}

func TestIsAdmin_PassesThrough(t *testing.T) {
	repo := &fakeRepo{isAdmin: true}
	svc := &Service{Repo: repo}

	isAdmin, err := svc.IsAdmin(context.Background(), "u1")
	if err != nil || !isAdmin {
		t.Fatalf("expected IsAdmin to pass through as true, got %v %v", isAdmin, err)
	}
}
