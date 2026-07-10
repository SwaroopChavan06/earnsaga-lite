package analytics

import (
	"encoding/json"
	"net/http"

	"earnsaga-lite/internal/auth"
	"earnsaga-lite/internal/common"
)

type trackRequest struct {
	Type    string `json:"type"`
	OfferID string `json:"offer_id"`
}

type Handler struct {
	Service *Service
}

// TrackEvent is the public /api/v1/events ingestion endpoint — any
// authenticated user can fire impression/click events from the frontend.
func (h *Handler) TrackEvent(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	var req trackRequest
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

// GetAnalytics is the admin-gated /admin/analytics reporting endpoint (see
// auth.RequireAdmin in main.go for the gate). Accepts optional ?from=&to=
// (YYYY-MM-DD, defaulting to the last 30 days) and ?offer_id= filters, and
// returns metrics grouped by date and offer per the assignment spec.
func (h *Handler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	offerID := q.Get("offer_id")
	if !ValidOfferID(offerID) {
		common.WriteError(w, http.StatusBadRequest, "offer_id must be a valid UUID")
		return
	}

	from, to, err := ParseRange(q.Get("from"), q.Get("to"))
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	report, err := h.Service.GetReport(r.Context(), from, to, offerID)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to fetch analytics")
		return
	}
	common.WriteJSON(w, http.StatusOK, report)
}
