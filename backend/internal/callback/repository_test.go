package callback

import (
	"testing"
	"time"
)

func f64ptr(v float64) *float64 { return &v }
func strptr(v string) *string   { return &v }

func TestSelectAttributionMatch(t *testing.T) {
	now := time.Now()

	t.Run("no in-progress offers at all", func(t *testing.T) {
		got := selectAttributionMatch(nil, 25.0)
		if got.userOfferID != nil || got.offerID != nil || got.goalID != nil {
			t.Fatalf("expected an entirely empty match, got %+v", got)
		}
	})

	t.Run("single candidate, reward matches exactly", func(t *testing.T) {
		candidates := []offerCandidate{
			{userOfferID: "uo-1", offerID: "off-1", startedAt: now, goalID: strptr("goal-1"), goalReward: f64ptr(25.0)},
		}
		got := selectAttributionMatch(candidates, 25.0)
		if got.offerID == nil || *got.offerID != "off-1" {
			t.Fatalf("expected offer off-1, got %+v", got)
		}
		if got.goalID == nil || *got.goalID != "goal-1" {
			t.Fatalf("expected goal goal-1 to be attributed, got %+v", got)
		}
	})

	t.Run("reward match wins even when it is not the oldest offer", func(t *testing.T) {
		candidates := []offerCandidate{
			{userOfferID: "uo-old", offerID: "off-old", startedAt: now, goalID: strptr("goal-old"), goalReward: f64ptr(999.0)},
			{userOfferID: "uo-new", offerID: "off-new", startedAt: now.Add(time.Hour), goalID: strptr("goal-new"), goalReward: f64ptr(25.0)},
		}
		got := selectAttributionMatch(candidates, 25.0)
		if got.offerID == nil || *got.offerID != "off-new" {
			t.Fatalf("expected the offer whose goal reward matches (off-new), got %+v", got)
		}
	})

	t.Run("no reward matches -> falls back to oldest in-progress offer", func(t *testing.T) {
		candidates := []offerCandidate{
			{userOfferID: "uo-1", offerID: "off-1", startedAt: now, goalID: strptr("goal-1"), goalReward: f64ptr(10.0)},
			{userOfferID: "uo-2", offerID: "off-2", startedAt: now.Add(time.Hour), goalID: strptr("goal-2"), goalReward: f64ptr(15.0)},
		}
		got := selectAttributionMatch(candidates, 999.0)
		if got.offerID == nil || *got.offerID != "off-1" {
			t.Fatalf("expected fallback to the oldest offer (off-1), got %+v", got)
		}
		if got.goalID != nil {
			t.Fatalf("expected no specific goal attributed on a value-mismatch fallback, got %+v", got)
		}
	})

	t.Run("offer with no goals still falls back cleanly", func(t *testing.T) {
		candidates := []offerCandidate{
			{userOfferID: "uo-1", offerID: "off-1", startedAt: now, goalID: nil, goalReward: nil},
		}
		got := selectAttributionMatch(candidates, 25.0)
		if got.offerID == nil || *got.offerID != "off-1" {
			t.Fatalf("expected fallback to off-1 even with no goals configured, got %+v", got)
		}
		if got.goalID != nil {
			t.Fatalf("expected nil goal for a goal-less offer, got %+v", got)
		}
	})

	t.Run("reward comparison tolerates float rounding", func(t *testing.T) {
		candidates := []offerCandidate{
			{userOfferID: "uo-1", offerID: "off-1", startedAt: now, goalID: strptr("goal-1"), goalReward: f64ptr(19.99)},
		}
		got := selectAttributionMatch(candidates, 19.990000001) // float noise from JSON/DB round-tripping
		if got.goalID == nil || *got.goalID != "goal-1" {
			t.Fatalf("expected reward match despite tiny float noise, got %+v", got)
		}
	})
}
