package handlers

import (
	"net/http"
	"time"

	"earnsaga-lite/internal/auth"
	"earnsaga-lite/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserHandler struct {
	DB  *pgxpool.Pool
	Cfg *config.Config
}

type userSummary struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	IsAdmin   bool   `json:"is_admin"`
}

type transaction struct {
	ID        string    `json:"id"`
	Amount    float64   `json:"amount"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

// GetProfile returns the currently authenticated user's profile.
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var u userSummary
	err := h.DB.QueryRow(r.Context(), `
		SELECT id, email, name, avatar_url, is_admin FROM users WHERE id = $1
	`, userID).Scan(&u.ID, &u.Email, &u.Name, &u.AvatarURL, &u.IsAdmin)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, u)
}

// GetWallet returns the user's current balance.
func (h *UserHandler) GetWallet(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var balance float64
	err := h.DB.QueryRow(r.Context(), `
		SELECT COALESCE(SUM(amount), 0) FROM wallet_transactions WHERE user_id = $1
	`, userID).Scan(&balance)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch balance")
		return
	}

	writeJSON(w, http.StatusOK, map[string]float64{"balance": balance})
}

// GetTransactions returns the user's transaction history.
func (h *UserHandler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	rows, err := h.DB.Query(r.Context(), `
		SELECT id, amount, type, created_at FROM wallet_transactions WHERE user_id = $1 ORDER BY created_at DESC
	`, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch transactions")
		return
	}
	defer rows.Close()

	var txs []transaction
	for rows.Next() {
		var t transaction
		if err := rows.Scan(&t.ID, &t.Amount, &t.Type, &t.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to parse transactions")
			return
		}
		txs = append(txs, t)
	}

	writeJSON(w, http.StatusOK, txs)
}
