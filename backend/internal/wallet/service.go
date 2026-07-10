package wallet

import (
	"context"
	"time"
)

type WalletBalanceResponse struct {
	Balance float64 `json:"balance"`
}

// TransactionResponse includes the offer/goal reference the assignment
// asks for. OfferID/GoalID/OfferName are omitted from the JSON entirely
// (not just null) when a transaction couldn't be attributed to an offer —
// e.g. a callback that arrived with no matching in-progress offer.
type TransactionResponse struct {
	ID        string    `json:"id"`
	Amount    float64   `json:"amount"`
	Type      string    `json:"type"`
	OfferID   *string   `json:"offer_id,omitempty"`
	GoalID    *string   `json:"goal_id,omitempty"`
	OfferName *string   `json:"offer_name,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// repository is the seam wallet.Service depends on. Unexported since it
// only exists to let tests substitute a fake — the concrete *Repository
// (in repository.go) satisfies it implicitly, so main.go wiring is
// untouched.
type repository interface {
	GetBalance(ctx context.Context, userID string) (float64, error)
	GetTransactions(ctx context.Context, userID string) ([]TransactionRow, error)
}

type Service struct {
	Repo repository
}

func (s *Service) GetBalance(ctx context.Context, userID string) (*WalletBalanceResponse, error) {
	balance, err := s.Repo.GetBalance(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &WalletBalanceResponse{Balance: balance}, nil
}

func (s *Service) GetTransactions(ctx context.Context, userID string) ([]TransactionResponse, error) {
	txs, err := s.Repo.GetTransactions(ctx, userID)
	if err != nil {
		return nil, err
	}
	response := make([]TransactionResponse, 0, len(txs))
	for _, t := range txs {
		response = append(response, TransactionResponse{
			ID:        t.ID,
			Amount:    t.Amount,
			Type:      t.Type,
			OfferID:   t.OfferID,
			GoalID:    t.GoalID,
			OfferName: t.OfferName,
			CreatedAt: t.CreatedAt,
		})
	}
	return response, nil
}
