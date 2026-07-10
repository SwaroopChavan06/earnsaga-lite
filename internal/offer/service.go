package offer

import (
	"context"
	"errors"
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
	ListActive(ctx context.Context, search string) ([]models.Offer, error)
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

func (s *Service) List(ctx context.Context, search string) ([]models.Offer, error) {
	return s.Repo.ListActive(ctx, strings.TrimSpace(search))
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

// Start implements the "Start Offer" click. Never errors on a repeat
// click — per spec, it returns the current state instead. The user ID is
// substituted into PubScale's {your_user_id} placeholder in trk_url, and
// that same ID is what PubScale will echo back in the S2S callback.
func (s *Service) Start(ctx context.Context, offerID, userID string) (*StartResult, error) {
	o, err := s.Repo.GetByID(ctx, offerID)
	if err != nil {
		return nil, err
	}
	redirectURL := strings.ReplaceAll(o.TrackingURL, "{your_user_id}", userID)

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
