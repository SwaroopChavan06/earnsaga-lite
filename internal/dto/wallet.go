package dto

import "time"

type WalletBalanceResponse struct {
	Balance float64 `json:"balance"`
}

type TransactionResponse struct {
	ID        string    `json:"id"`
	Amount    float64   `json:"amount"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}
