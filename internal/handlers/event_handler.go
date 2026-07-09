package handlers

import (
	"encoding/json"
	"net/http"

	"earnsaga-lite/internal/auth"
	"earnsaga-lite/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

type EventHandler struct {
	DB  *pgxpool.Pool
	Cfg *config.Config
}

type eventRequest struct {
	Type    string `json:"type"` // impression | click
	OfferID string `json:"offer_id"`
}

// TrackEvent records an impression or click event for a user.
func (h *EventHandler) TrackEvent(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req eventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Type != "impression" && req.Type != "click" {
		writeError(w, http.StatusBadRequest, "invalid event type")
		return
	}

	_, err := h.DB.Exec(r.Context(), `
		INSERT INTO events (type, offer_id, user_id)
		VALUES ($1, $2, $3)
	`, req.Type, req.OfferID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to track event")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"status": "tracked"})
}
