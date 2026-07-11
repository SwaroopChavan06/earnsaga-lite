package wallet

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeRepo struct {
	balance float64
	txs     []TransactionRow

	balanceErr error
	txsErr     error
}

func (f *fakeRepo) GetBalance(ctx context.Context, userID string) (float64, error) {
	return f.balance, f.balanceErr
}
func (f *fakeRepo) GetTransactions(ctx context.Context, userID string) ([]TransactionRow, error) {
	return f.txs, f.txsErr
}

func TestGetSummary_CombinesBalanceAndTransactions(t *testing.T) {
	offerName := "Test Offer"
	repo := &fakeRepo{
		balance: 42.5,
		txs: []TransactionRow{
			{ID: "tx-1", Amount: 10, Type: "credit", OfferName: &offerName, CreatedAt: time.Now()},
		},
	}
	svc := &Service{Repo: repo}

	summary, err := svc.GetSummary(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.BalanceUSD != 42.5 {
		t.Fatalf("expected balance 42.5, got %v", summary.BalanceUSD)
	}
	if len(summary.Transactions) != 1 || summary.Transactions[0].ID != "tx-1" {
		t.Fatalf("expected 1 transaction with ID tx-1, got %+v", summary.Transactions)
	}
}

func TestGetSummary_PropagatesBalanceError(t *testing.T) {
	repo := &fakeRepo{balanceErr: errors.New("db down")}
	svc := &Service{Repo: repo}

	if _, err := svc.GetSummary(context.Background(), "user-1"); err == nil {
		t.Fatal("expected an error when the balance query fails")
	}
}

func TestGetSummary_PropagatesTransactionsError(t *testing.T) {
	repo := &fakeRepo{txsErr: errors.New("db down")}
	svc := &Service{Repo: repo}

	if _, err := svc.GetSummary(context.Background(), "user-1"); err == nil {
		t.Fatal("expected an error when the transactions query fails")
	}
}
