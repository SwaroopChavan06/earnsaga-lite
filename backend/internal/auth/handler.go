package auth

import (
	"context"
	"encoding/json"
	"net/http"

	"earnsaga-lite/internal/common"
	"earnsaga-lite/internal/config"
	"earnsaga-lite/internal/models"
)

// UserUpserter is the only thing Handler needs from the user domain. It's
// defined here (over the shared models package) rather than importing
// user.Service directly, because user.Handler already imports auth for
// context extraction — importing user back from auth would be a cycle.
// *user.Service satisfies this interface structurally.
type UserUpserter interface {
	FindOrCreateByGoogle(ctx context.Context, sub, email, name, avatarURL string) (*models.User, error)
}

type Handler struct {
	UserService UserUpserter
	Cfg         *config.Config
}

type googleLoginRequest struct {
	IDToken string `json:"id_token"`
}

type loginResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

// GoogleLogin verifies the Google id_token from the frontend, finds-or-creates
// the user via UserService (user creation is that domain's responsibility,
// not auth's), and returns our own JWT.
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

	u, err := h.UserService.FindOrCreateByGoogle(r.Context(), gUser.Sub, gUser.Email, gUser.Name, gUser.Picture)
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

	u, err := h.UserService.FindOrCreateByGoogle(r.Context(), "dev-sub-"+email, email, "Dev User", "")
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
