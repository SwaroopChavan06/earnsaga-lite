package offer

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"

	"earnsaga-lite/internal/models"
	"earnsaga-lite/internal/pubscale"

	"github.com/jackc/pgx/v5"
	"golang.org/x/sync/errgroup"
)

// repository is the seam offer.Service depends on. Unexported since it
// only exists to let tests substitute a fake — the concrete *Repository
// (in repository.go) satisfies it implicitly, so main.go wiring is
// untouched.
type repository interface {
	UpsertFromPubScale(ctx context.Context, o pubscale.Offer) error
	ListActive(ctx context.Context, search string, limit, offset int) ([]models.Offer, int, error)
	GetByID(ctx context.Context, id string) (*models.Offer, error)
	ListGoals(ctx context.Context, offerID string) ([]models.OfferGoal, error)
	GetUserOfferStatus(ctx context.Context, userID, offerID string) (string, error)
	InsertStart(ctx context.Context, userID, offerID string) error
}

type Service struct {
	Repo     repository
	PubScale *pubscale.Client
}

// syncWorkers is the number of parallel DB upserts during a PubScale sync.
// High enough to saturate the connection pool but low enough not to stampede it.
const syncWorkers = 10

// SyncFromPubScale pages through the full PubScale catalog and upserts
// everything. Deliberately called with a context that has its own generous
// timeout rather than the HTTP request's — thousands of offers can
// legitimately take longer than a typical request timeout.
//
// Each page of offers is fanned out to a bounded pool of syncWorkers
// goroutines so upserts happen in parallel — total time is roughly
// (pages × PubScale RTT) + (max_page_size / syncWorkers × DB RTT) instead
// of (pages × PubScale RTT) + (total_offers × DB RTT).
func (s *Service) SyncFromPubScale(ctx context.Context) (int, error) {
	const pageSize = 200

	var totalSynced atomic.Int64
	page := 1

	for {
		resp, err := s.PubScale.FetchOffers(page, pageSize)
		if err != nil {
			return int(totalSynced.Load()), err
		}
		if len(resp.Offers) == 0 {
			break
		}

		// Fan out upserts for this page across syncWorkers goroutines.
		jobs := make(chan pubscale.Offer, len(resp.Offers))
		for _, o := range resp.Offers {
			jobs <- o
		}
		close(jobs)

		var wg sync.WaitGroup
		for range syncWorkers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for o := range jobs {
					if ctx.Err() != nil {
						return
					}
					if err := s.Repo.UpsertFromPubScale(ctx, o); err != nil {
						// One bad offer must not abort the batch; caller sees
						// totalSynced < resp.Total as an indicator of partial failure.
						continue
					}
					totalSynced.Add(1)
				}
			}()
		}
		wg.Wait()

		if len(resp.Offers) < pageSize || int(totalSynced.Load()) >= resp.Total {
			break
		}
		page++
	}

	return int(totalSynced.Load()), nil
}

const DefaultPageLimit = 20

// ListPage is the paginated response for the offers list endpoint.
type ListPage struct {
	Offers []models.Offer `json:"offers"`
	Total  int            `json:"total"`
	Page   int            `json:"page"`
	Limit  int            `json:"limit"`
	Pages  int            `json:"pages"`
}

// List returns one page of active offers. page and limit are 1-indexed and
// clamped: page >= 1, 1 <= limit <= 100.
func (s *Service) List(ctx context.Context, search string, page, limit int) (*ListPage, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = DefaultPageLimit
	}
	offset := (page - 1) * limit

	offers, total, err := s.Repo.ListActive(ctx, strings.TrimSpace(search), limit, offset)
	if err != nil {
		return nil, err
	}

	pages := total / limit
	if total%limit != 0 {
		pages++
	}
	if pages < 1 {
		pages = 1
	}

	return &ListPage{
		Offers: offers,
		Total:  total,
		Page:   page,
		Limit:  limit,
		Pages:  pages,
	}, nil
}

type Detail struct {
	models.Offer
	Goals  []models.OfferGoal `json:"goals"`
	Status string             `json:"status"` // not_started | in_progress | completed
}

// GetDetail returns the offer, its goals, and the calling user's current
// status on it — this drives the Start Offer / In Progress button.
//
// After fetching the offer itself, ListGoals and GetUserOfferStatus are
// independent DB calls — they run in parallel via errgroup so latency is
// max(goals, status) instead of goals+status.
func (s *Service) GetDetail(ctx context.Context, offerID, userID string) (*Detail, error) {
	o, err := s.Repo.GetByID(ctx, offerID)
	if err != nil {
		return nil, err
	}

	var goals []models.OfferGoal
	status := string(models.StatusNotStarted)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		goals, err = s.Repo.ListGoals(gctx, offerID)
		return err
	})

	if userID != "" {
		g.Go(func() error {
			st, err := s.Repo.GetUserOfferStatus(gctx, userID, offerID)
			if err == nil {
				status = st
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return err // real DB error; ErrNoRows means "not started" — not an error
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return &Detail{Offer: *o, Goals: goals, Status: status}, nil
}

type StartResult struct {
	Status         string `json:"status"`
	RedirectURL    string `json:"redirect_url"`
	AlreadyStarted bool   `json:"already_started"`
}

// buildRedirectURL substitutes PubScale's {your_user_id} placeholder with
// the real user ID, then drops gaid/idfa if they're still the raw
// unresolved placeholders ("{gaid_for_android}" / "{idfa_for_ios}").
// PubScale's docs list those as recommended, mobile-only device
// identifiers (Google Advertising ID / IDFA) — we're a web client and
// have no real value to put there, so we omit them entirely rather than
// forward literal template syntax as if it were a real identifier.
func buildRedirectURL(trackingURL, userID string) string {
	substituted := strings.ReplaceAll(trackingURL, "{your_user_id}", userID)

	u, err := url.Parse(substituted)
	if err != nil {
		return substituted // malformed URL is unexpected; pass through rather than fail Start()
	}

	q := u.Query()
	for _, param := range []string{"gaid", "idfa"} {
		if strings.Contains(q.Get(param), "{") {
			q.Del(param)
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// Start implements the "Start Offer" click. Never errors on a repeat
// click — per spec, it returns the current state instead. The user ID is
// substituted into PubScale's {your_user_id} placeholder in trk_url, and
// that same ID is what PubScale will echo back in the S2S callback.
func (s *Service) Start(ctx context.Context, offerID, userID string) (*StartResult, error) {
	o, err := s.Repo.GetByID(ctx, offerID)
	if err != nil {
		return nil, err
	}
	redirectURL := buildRedirectURL(o.TrackingURL, userID)

	existingStatus, err := s.Repo.GetUserOfferStatus(ctx, userID, offerID)
	if err == nil {
		return &StartResult{Status: existingStatus, RedirectURL: redirectURL, AlreadyStarted: true}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err // real DB error
	}

	if err := s.Repo.InsertStart(ctx, userID, offerID); err != nil {
		return nil, err
	}

	return &StartResult{Status: string(models.StatusInProgress), RedirectURL: redirectURL, AlreadyStarted: false}, nil
}
