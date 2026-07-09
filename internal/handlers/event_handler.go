package handlers

import (
	"encoding/json"
	"net/http"

	"earnsaga-lite/internal/auth"
	"earnsaga-lite/internal/services"
)

type EventHandler struct {
	Service *services.EventService
}

type eventRequest struct {
	Type    string `json:"type"` // impression | click
	OfferID string `json:"offer_id"`
}

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

	if err := h.Service.Track(r.Context(), userID, req.OfferID, req.Type); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to track event")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"status": "tracked"})
}
