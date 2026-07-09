package services

import (
	"context"
	"earnsaga-lite/internal/dto"
	"earnsaga-lite/internal/repositories"
)

type WalletService struct {
	Repo *repositories.WalletRepository
}

func (s *WalletService) GetBalance(ctx context.Context, userID string) (*dto.WalletBalanceResponse, error) {
	balance, err := s.Repo.GetBalance(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &dto.WalletBalanceResponse{Balance: balance}, nil
}

func (s *WalletService) GetTransactions(ctx context.Context, userID string) ([]dto.TransactionResponse, error) {
	txs, err := s.Repo.GetTransactions(ctx, userID)
	if err != nil {
		return nil, err
	}

	var response []dto.TransactionResponse
	for _, t := range txs {
		response = append(response, dto.TransactionResponse{
			ID:        t.ID,
			Amount:    t.Amount,
			Type:      t.Type,
			CreatedAt: t.CreatedAt,
		})
	}
	return response, nil
}
