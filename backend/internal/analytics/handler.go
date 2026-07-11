package analytics

import (
	"encoding/json"
	"net/http"
	"strconv"

	"earnsaga-lite/internal/auth"
	"earnsaga-lite/internal/common"
)

type trackRequest struct {
	Type    string `json:"type"`
	OfferID string `json:"offer_id"`
}

// batchTrackRequest mirrors trackRequest but for the bulk ingestion
// endpoint. Timestamp is accepted for forward-compatibility with the
// frontend payload but intentionally not used to override created_at —
// the server's own clock is the single source of truth for ordering.
type batchTrackRequest struct {
	Events []struct {
		Type      string `json:"type"`
		OfferID   string `json:"offer_id"`
		Timestamp string `json:"timestamp"`
	} `json:"events"`
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

// TrackEventBatch is the bulk /api/v1/events/batch ingestion endpoint. The
// frontend queues impressions/clicks client-side and flushes them here
// periodically instead of firing one HTTP request per event — a page of
// 20 offer cards previously meant 20 individual requests and DB inserts.
//
// Kept as an additive route alongside TrackEvent (unchanged) so existing
// single-event callers are unaffected.
func (h *Handler) TrackEventBatch(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req batchTrackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.Events) == 0 {
		common.WriteError(w, http.StatusBadRequest, "events must not be empty")
		return
	}

	events := make([]BatchEvent, len(req.Events))
	for i, e := range req.Events {
		if e.Type != "impression" && e.Type != "click" {
			common.WriteError(w, http.StatusBadRequest, "invalid event type at index "+strconv.Itoa(i))
			return
		}
		if !ValidOfferID(e.OfferID) || e.OfferID == "" {
			common.WriteError(w, http.StatusBadRequest, "offer_id must be a valid UUID at index "+strconv.Itoa(i))
			return
		}
		events[i] = BatchEvent{Type: e.Type, OfferID: e.OfferID}
	}

	if err := h.Service.TrackBatch(r.Context(), userID, events); err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to track events")
		return
	}
	common.WriteJSON(w, http.StatusCreated, map[string]int{"accepted": len(events)})
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
