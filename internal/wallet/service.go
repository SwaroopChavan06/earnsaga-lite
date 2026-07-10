package wallet

import (
	"context"
	"time"

	"earnsaga-lite/internal/models"
)

type WalletBalanceResponse struct {
	Balance float64 `json:"balance"`
}

type TransactionResponse struct {
	ID        string    `json:"id"`
	Amount    float64   `json:"amount"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

// repository is the seam wallet.Service depends on. Unexported since it
// only exists to let tests substitute a fake — the concrete *Repository
// (in repository.go) satisfies it implicitly, so main.go wiring is
// untouched.
type repository interface {
	GetBalance(ctx context.Context, userID string) (float64, error)
	GetTransactions(ctx context.Context, userID string) ([]models.WalletTransaction, error)
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
	var response []TransactionResponse
	for _, t := range txs {
		response = append(response, TransactionResponse{
			ID:        t.ID,
			Amount:    t.Amount,
			Type:      t.Type,
			CreatedAt: t.CreatedAt,
		})
	}
	return response, nil
}
