package analytics

import (
	"context"
	"testing"
	"time"
)

func TestParseRange_DefaultsToLast30Days(t *testing.T) {
	from, to, err := ParseRange("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff := to.Sub(from); diff != 30*24*time.Hour {
		t.Fatalf("expected a 30-day window when unset, got %v", diff)
	}
	if !to.After(time.Now().UTC()) {
		t.Fatalf("expected the default 'to' to include all of today (be in the future), got %v", to)
	}
}

func TestParseRange_ExplicitRangeIsHalfOpen(t *testing.T) {
	from, to, err := ParseRange("2024-01-01", "2024-01-10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	wantTo := time.Date(2024, 1, 11, 0, 0, 0, 0, time.UTC) // exclusive, one day past "to" so Jan 10 is fully included
	if !from.Equal(wantFrom) {
		t.Fatalf("expected from=%v, got %v", wantFrom, from)
	}
	if !to.Equal(wantTo) {
		t.Fatalf("expected to=%v (exclusive), got %v", wantTo, to)
	}
}

func TestParseRange_InvalidFormatRejected(t *testing.T) {
	if _, _, err := ParseRange("01/01/2024", ""); err == nil {
		t.Fatal("expected a non-YYYY-MM-DD 'from' to be rejected")
	}
	if _, _, err := ParseRange("", "not-a-date"); err == nil {
		t.Fatal("expected a non-YYYY-MM-DD 'to' to be rejected")
	}
}

func TestParseRange_FromAfterToRejected(t *testing.T) {
	_, _, err := ParseRange("2024-02-01", "2024-01-01")
	if err == nil {
		t.Fatal("expected an error when from is after to")
	}
}

func TestValidOfferID(t *testing.T) {
	cases := []struct {
		id   string
		want bool
	}{
		{"", true},
		{"123e4567-e89b-12d3-a456-426614174000", true},
		{"not-a-uuid", false},
		{"123e4567e89b12d3a456426614174000", false}, // missing dashes
		{"'; DROP TABLE offers; --", false},
	}
	for _, c := range cases {
		if got := ValidOfferID(c.id); got != c.want {
			t.Errorf("ValidOfferID(%q) = %v, want %v", c.id, got, c.want)
		}
	}
}

type fakeRepo struct {
	createCalls int
	lastType    string
	report      *Report
}

func (f *fakeRepo) Create(ctx context.Context, userID, offerID, eventType string) error {
	f.createCalls++
	f.lastType = eventType
	return nil
}
func (f *fakeRepo) GetReport(ctx context.Context, from, to time.Time, offerID string) (*Report, error) {
	return f.report, nil
}

func TestTrack_PassesThroughToRepository(t *testing.T) {
	repo := &fakeRepo{}
	svc := &Service{Repo: repo}

	if err := svc.Track(context.Background(), "user-1", "offer-1", "click"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.createCalls != 1 || repo.lastType != "click" {
		t.Fatalf("expected Track to forward the event to the repository, got calls=%d type=%q", repo.createCalls, repo.lastType)
	}
}
