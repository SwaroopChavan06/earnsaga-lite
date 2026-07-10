package offer

import (
	"context"
	"errors"
	"strings"

	"earnsaga-lite/internal/models"
	"earnsaga-lite/internal/pubscale"

	"github.com/jackc/pgx/v5"
)

type Service struct {
	Repo     *Repository
	PubScale *pubscale.Client
}

// SyncFromPubScale pages through the full PubScale catalog and upserts
// everything. Deliberately called with a context that has its own
// generous timeout (set by the caller/handler) rather than the HTTP
// request's — thousands of offers, each in its own transaction, can
// legitimately take longer than a typical request timeout.
func (s *Service) SyncFromPubScale(ctx context.Context) (int, error) {
	const pageSize = 200
	totalSynced := 0
	page := 1

	for {
		resp, err := s.PubScale.FetchOffers(page, pageSize)
		if err != nil {
			return totalSynced, err
		}
		if len(resp.Offers) == 0 {
			break
		}

		for _, o := range resp.Offers {
			if err := s.Repo.UpsertFromPubScale(ctx, o); err != nil {
				// Don't let one bad offer abort the whole sync — log at the
				// handler/caller level and keep going.
				continue
			}
			totalSynced++
		}

		if len(resp.Offers) < pageSize || totalSynced >= resp.Total {
			break
		}
		page++
	}

	return totalSynced, nil
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
func (s *Service) GetDetail(ctx context.Context, offerID, userID string) (*Detail, error) {
	o, err := s.Repo.GetByID(ctx, offerID)
	if err != nil {
		return nil, err
	}

	goals, err := s.Repo.ListGoals(ctx, offerID)
	if err != nil {
		return nil, err
	}

	status := string(models.StatusNotStarted)
	if userID != "" {
		st, err := s.Repo.GetUserOfferStatus(ctx, userID, offerID)
		if err == nil {
			status = st
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err // a real DB error, not just "hasn't started"
		}
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
