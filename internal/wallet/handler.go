package wallet

import (
	"net/http"

	"earnsaga-lite/internal/auth"
	"earnsaga-lite/internal/common"
)

type Handler struct {
	Service *Service
}

func (h *Handler) GetWallet(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	balance, err := h.Service.GetBalance(r.Context(), userID)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to fetch balance")
		return
	}

	common.WriteJSON(w, http.StatusOK, balance)
}

func (h *Handler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	txs, err := h.Service.GetTransactions(r.Context(), userID)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to fetch transactions")
		return
	}

	common.WriteJSON(w, http.StatusOK, txs)
}
