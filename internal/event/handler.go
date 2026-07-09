package event

import (
	"encoding/json"
	"net/http"

	"earnsaga-lite/internal/auth"
	"earnsaga-lite/internal/common"
)

type Handler struct {
	Service *Service
}

type request struct {
	Type    string `json:"type"` // impression | click
	OfferID string `json:"offer_id"`
}

func (h *Handler) TrackEvent(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Type != "impression" && req.Type != "click" {
		common.WriteError(w, http.StatusBadRequest, "invalid event type")
		return
	}

	if err := h.Service.Track(r.Context(), userID, req.OfferID, req.Type); err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to track event")
		return
	}

	common.WriteJSON(w, http.StatusCreated, map[string]string{"status": "tracked"})
}
