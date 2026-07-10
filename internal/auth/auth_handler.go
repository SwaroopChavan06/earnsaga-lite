package auth

import (
	"encoding/json"
	"net/http"

	"earnsaga-lite/internal/common"
	"earnsaga-lite/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	DB  *pgxpool.Pool
	Cfg *config.Config
}

type googleLoginRequest struct {
	IDToken string `json:"id_token"`
}

type userSummary struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	IsAdmin   bool   `json:"is_admin"`
}

type loginResponse struct {
	Token string      `json:"token"`
	User  userSummary `json:"user"`
}

// GoogleLogin verifies the Google id_token from the frontend, finds-or-creates
// the user, and returns our own JWT. This is the route that was missing after
// the refactor — the auth package had the building blocks (VerifyGoogleIDToken,
// IssueToken) but nothing wired them to an HTTP route.
func (h *Handler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	var req googleLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.IDToken == "" {
		common.WriteError(w, http.StatusBadRequest, "id_token is required")
		return
	}

	gUser, err := VerifyGoogleIDToken(r.Context(), h.Cfg.GoogleClientID, req.IDToken)
	if err != nil {
		common.WriteError(w, http.StatusUnauthorized, "google token verification failed")
		return
	}

	var u userSummary
	err = h.DB.QueryRow(r.Context(), `
		INSERT INTO users (google_sub, email, name, avatar_url)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (google_sub) DO UPDATE SET email = EXCLUDED.email
		RETURNING id, email, name, avatar_url, is_admin
	`, gUser.Sub, gUser.Email, gUser.Name, gUser.Picture).
		Scan(&u.ID, &u.Email, &u.Name, &u.AvatarURL, &u.IsAdmin)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to create or fetch user")
		return
	}

	token, err := IssueToken(h.Cfg.JWTSecret, u.ID, u.Email)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	common.WriteJSON(w, http.StatusOK, loginResponse{Token: token, User: u})
}

// IssueDevToken mints a JWT without going through Google OAuth — only ever
// wired in main.go when Cfg.Env == "development", so you can curl every
// protected route before the frontend's login flow exists.
func (h *Handler) IssueDevToken(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		email = "dev@example.com"
	}

	var u userSummary
	err := h.DB.QueryRow(r.Context(), `
		INSERT INTO users (google_sub, email, name, avatar_url)
		VALUES ($1, $2, $3, '')
		ON CONFLICT (google_sub) DO UPDATE SET email = EXCLUDED.email
		RETURNING id, email, name, avatar_url, is_admin
	`, "dev-sub-"+email, email, "Dev User").
		Scan(&u.ID, &u.Email, &u.Name, &u.AvatarURL, &u.IsAdmin)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to create dev user")
		return
	}

	token, err := IssueToken(h.Cfg.JWTSecret, u.ID, u.Email)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	common.WriteJSON(w, http.StatusOK, loginResponse{Token: token, User: u})
}
