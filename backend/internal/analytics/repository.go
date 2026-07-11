package analytics

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
)

type Repository struct {
	DB *pgxpool.Pool
}

// DateOfferRow is one (date, offer) bucket of aggregated metrics.
// OfferID/OfferName are nil for the bucket of events/revenue that couldn't
// be tied to any offer (e.g. an unattributed wallet credit).
type DateOfferRow struct {
	Date        string  `json:"date"`
	OfferID     *string `json:"offer_id,omitempty"`
	OfferName   *string `json:"offer_name,omitempty"`
	Impressions int     `json:"impressions"`
	Clicks      int     `json:"clicks"`
	Revenue     float64 `json:"revenue"`
}

// DAURow is one day's distinct active user count. DAU isn't offer-scoped,
// so it's its own per-date series rather than folded into DateOfferRow
// (which would force an artificial offer dimension onto a platform-wide
// metric).
type DAURow struct {
	Date string `json:"date"`
	DAU  int    `json:"dau"`
}

type Report struct {
	ByDateOffer []DateOfferRow `json:"by_date_offer"`
	ByDate      []DAURow       `json:"by_date"`
}

// Create inserts one tracking event (impression | click) fired by the
// frontend. offer_id/user_id are nullable at the DB level (ON DELETE SET
// NULL), so this never fails due to a since-deleted offer or user.
func (r *Repository) Create(ctx context.Context, userID, offerID, eventType string) error {
	_, err := r.DB.Exec(ctx, `
		INSERT INTO events (type, offer_id, user_id)
		VALUES ($1, $2, $3)
	`, eventType, offerID, userID)
	return err
}

// BatchEvent is one tracking event within a batch ingestion request.
type BatchEvent struct {
	Type    string
	OfferID string
	UserID  string
}

// CreateBatch inserts any number of tracking events in a single round-trip
// using unnest() to expand three parallel arrays into rows — this is the
// production-grade way to ingest high-frequency events (impressions fire
// per offer card, so a page of 20 offers previously meant 20 individual
// INSERTs). Cost is O(1) DB round-trips regardless of batch size.
func (r *Repository) CreateBatch(ctx context.Context, events []BatchEvent) error {
	types := make([]string, len(events))
	offerIDs := make([]string, len(events))
	userIDs := make([]string, len(events))
	for i, e := range events {
		types[i] = e.Type
		offerIDs[i] = e.OfferID
		userIDs[i] = e.UserID
	}

	_, err := r.DB.Exec(ctx, `
		INSERT INTO events (type, offer_id, user_id)
		SELECT * FROM unnest($1::text[], $2::uuid[], $3::uuid[])
	`, types, offerIDs, userIDs)
	return err
}

// GetReport aggregates impressions/clicks (from events) and revenue (from
// wallet_transactions) grouped by day and offer, plus a separate DAU series,
// over the half-open window [from, to). offerID is optional — pass "" to
// report across all offers.
//
// The two DB queries (by_date_offer and DAU) are fully independent — they run
// in parallel via errgroup so total latency is max(q1, q2) instead of q1+q2.
func (r *Repository) GetReport(ctx context.Context, from, to time.Time, offerID string) (*Report, error) {
	var offerFilter *string
	if offerID != "" {
		offerFilter = &offerID
	}

	var byDateOffer []DateOfferRow
	var byDate []DAURow

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		byDateOffer, err = r.getByDateOffer(gctx, from, to, offerFilter)
		return err
	})

	g.Go(func() error {
		var err error
		byDate, err = r.getDAU(gctx, from, to)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return &Report{ByDateOffer: byDateOffer, ByDate: byDate}, nil
}

func (r *Repository) getByDateOffer(ctx context.Context, from, to time.Time, offerFilter *string) ([]DateOfferRow, error) {
	rows, err := r.DB.Query(ctx, `
		WITH ev AS (
			SELECT date_trunc('day', created_at)::date AS day, offer_id,
			       COUNT(*) FILTER (WHERE type = 'impression') AS impressions,
			       COUNT(*) FILTER (WHERE type = 'click') AS clicks
			FROM events
			WHERE created_at >= $1 AND created_at < $2
			  AND ($3::uuid IS NULL OR offer_id = $3::uuid)
			GROUP BY 1, 2
		),
		rev AS (
			SELECT date_trunc('day', created_at)::date AS day, offer_id,
			       SUM(amount) AS revenue
			FROM wallet_transactions
			WHERE amount > 0
			  AND created_at >= $1 AND created_at < $2
			  AND ($3::uuid IS NULL OR offer_id = $3::uuid)
			GROUP BY 1, 2
		)
		SELECT
			COALESCE(ev.day, rev.day) AS day,
			COALESCE(ev.offer_id, rev.offer_id) AS offer_id,
			o.name,
			COALESCE(ev.impressions, 0),
			COALESCE(ev.clicks, 0),
			COALESCE(rev.revenue, 0)
		FROM ev
		FULL OUTER JOIN rev ON ev.day = rev.day AND ev.offer_id = rev.offer_id
		LEFT JOIN offers o ON o.id = COALESCE(ev.offer_id, rev.offer_id)
		ORDER BY day DESC
	`, from, to, offerFilter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []DateOfferRow{}
	for rows.Next() {
		var row DateOfferRow
		var day time.Time
		if err := rows.Scan(&day, &row.OfferID, &row.OfferName, &row.Impressions, &row.Clicks, &row.Revenue); err != nil {
			return nil, err
		}
		row.Date = day.Format("2006-01-02")
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *Repository) getDAU(ctx context.Context, from, to time.Time) ([]DAURow, error) {
	rows, err := r.DB.Query(ctx, `
		SELECT date_trunc('day', created_at)::date AS day, COUNT(DISTINCT user_id) AS dau
		FROM events
		WHERE created_at >= $1 AND created_at < $2 AND user_id IS NOT NULL
		GROUP BY 1
		ORDER BY 1 DESC
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []DAURow{}
	for rows.Next() {
		var row DAURow
		var day time.Time
		if err := rows.Scan(&day, &row.DAU); err != nil {
			return nil, err
		}
		row.Date = day.Format("2006-01-02")
		result = append(result, row)
	}
	return result, rows.Err()
}
