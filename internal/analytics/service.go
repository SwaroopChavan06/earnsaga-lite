package analytics

import "context"

// repository is the seam analytics.Service depends on. Unexported since it
// only exists to let tests substitute a fake — the concrete *Repository
// (in repository.go) satisfies it implicitly, so main.go wiring is
// untouched.
type repository interface {
	Create(ctx context.Context, userID, offerID, eventType string) error
	GetSummary(ctx context.Context) (*Summary, error)
}

type Service struct {
	Repo repository
}

func (s *Service) Track(ctx context.Context, userID, offerID, eventType string) error {
	return s.Repo.Create(ctx, userID, offerID, eventType)
}

func (s *Service) GetSummary(ctx context.Context) (*Summary, error) {
	return s.Repo.GetSummary(ctx)
}
