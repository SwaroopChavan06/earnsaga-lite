package handlers

import (
	"net/http"

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

// GetProfile returns the currently authenticated user's profile.
// This is mapped to GET /api/v1/users/profile.
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
