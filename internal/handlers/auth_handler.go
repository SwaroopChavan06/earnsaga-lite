package handlers

import (
	"encoding/json"
	"net/http"

	"earnsaga-lite/internal/auth"
	"earnsaga-lite/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthHandler struct {
	DB  *pgxpool.Pool
	Cfg *config.Config
}

type googleLoginRequest struct {
	IDToken string `json:"id_token"`
}

type loginResponse struct {
	Token string      `json:"token"`
	User  userSummary `json:"user"`
}

type userSummary struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	IsAdmin   bool   `json:"is_admin"`
}

// GoogleLogin verifies the Google id_token sent from the frontend,
// finds-or-creates the corresponding user row, and returns our own JWT.
func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	var req googleLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.IDToken == "" {
		writeError(w, http.StatusBadRequest, "id_token is required")
		return
	}

	gUser, err := auth.VerifyGoogleIDToken(r.Context(), h.Cfg.GoogleClientID, req.IDToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "google token verification failed")
		return
	}

	var u userSummary
	// find-or-create in one round trip: insert, on conflict just return existing row
	err = h.DB.QueryRow(r.Context(), `
		INSERT INTO users (google_sub, email, name, avatar_url)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (google_sub) DO UPDATE SET email = EXCLUDED.email
		RETURNING id, email, name, avatar_url, is_admin
	`, gUser.Sub, gUser.Email, gUser.Name, gUser.Picture).
		Scan(&u.ID, &u.Email, &u.Name, &u.AvatarURL, &u.IsAdmin)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create or fetch user")
		return
	}

	token, err := auth.IssueToken(h.Cfg.JWTSecret, u.ID, u.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{Token: token, User: u})
}
