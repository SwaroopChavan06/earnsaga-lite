package handlers

import (
	"net/http"

	"earnsaga-lite/internal/auth"
	"earnsaga-lite/internal/services"
)

type WalletHandler struct {
	Service *services.WalletService
}

func (h *WalletHandler) GetWallet(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	balance, err := h.Service.GetBalance(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch balance")
		return
	}

	writeJSON(w, http.StatusOK, balance)
}

func (h *WalletHandler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	txs, err := h.Service.GetTransactions(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch transactions")
		return
	}

	writeJSON(w, http.StatusOK, txs)
}
