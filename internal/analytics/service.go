package analytics

import (
	"context"
	"fmt"
	"regexp"
	"time"
)

// repository is the seam analytics.Service depends on. Unexported since it
// only exists to let tests substitute a fake — the concrete *Repository
// (in repository.go) satisfies it implicitly, so main.go wiring is
// untouched.
type repository interface {
	Create(ctx context.Context, userID, offerID, eventType string) error
	GetReport(ctx context.Context, from, to time.Time, offerID string) (*Report, error)
}

type Service struct {
	Repo repository
}

func (s *Service) Track(ctx context.Context, userID, offerID, eventType string) error {
	return s.Repo.Create(ctx, userID, offerID, eventType)
}

// GetReport expects an already-validated [from, to) window — see
// ParseRange, which the handler calls first so date-format errors surface
// as 400s before ever touching the database.
func (s *Service) GetReport(ctx context.Context, from, to time.Time, offerID string) (*Report, error) {
	return s.Repo.GetReport(ctx, from, to, offerID)
}

const dateLayout = "2006-01-02"

// ParseRange parses the from/to query params (YYYY-MM-DD), defaulting to
// the last 30 days when either is unset. Returns the half-open [from, to)
// window GetReport expects — "to" is exclusive, one day past the
// requested end date, so the requested end date's own events are still
// included. Pure (no I/O), so it's unit-testable on its own.
func ParseRange(fromStr, toStr string) (from, to time.Time, err error) {
	to = time.Now().UTC().Truncate(24 * time.Hour).AddDate(0, 0, 1) // tomorrow 00:00 UTC -- includes all of today
	from = to.AddDate(0, 0, -30)

	if toStr != "" {
		t, parseErr := time.Parse(dateLayout, toStr)
		if parseErr != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid to date, expected YYYY-MM-DD: %w", parseErr)
		}
		to = t.AddDate(0, 0, 1)
	}
	if fromStr != "" {
		t, parseErr := time.Parse(dateLayout, fromStr)
		if parseErr != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid from date, expected YYYY-MM-DD: %w", parseErr)
		}
		from = t
	}
	if from.After(to) {
		return time.Time{}, time.Time{}, fmt.Errorf("from date must not be after to date")
	}
	return from, to, nil
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ValidOfferID reports whether s is either empty (no filter) or looks like
// a UUID — checked before it ever reaches a SQL ::uuid cast, so a bad
// offer_id query param is a clean 400 instead of a raw DB error leaking out.
func ValidOfferID(s string) bool {
	return s == "" || uuidPattern.MatchString(s)
}
