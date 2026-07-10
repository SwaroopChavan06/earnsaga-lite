package offer

import (
	"context"
	"net/http"
	"time"

	"earnsaga-lite/internal/auth"
	"earnsaga-lite/internal/common"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	Service *Service
}

// SyncOffers is admin-gated (see main.go wiring) and deliberately does NOT
// use r.Context() — it builds its own long-lived context so the sync isn't
// cut short by the standard request timeout applied to other routes.
// Syncing thousands of offers one-by-one can take well over 30s.
func (h *Handler) SyncOffers(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	synced, err := h.Service.SyncFromPubScale(ctx)
	if err != nil {
		common.WriteError(w, http.StatusBadGateway, "failed to sync offers from pubscale")
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]int{"synced": synced})
}

func (h *Handler) ListOffers(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	offers, err := h.Service.List(r.Context(), search)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to fetch offers")
		return
	}
	common.WriteJSON(w, http.StatusOK, offers)
}

func (h *Handler) GetOfferDetail(w http.ResponseWriter, r *http.Request) {
	offerID := chi.URLParam(r, "id")
	userID, _ := auth.UserIDFromContext(r.Context()) // ok to be absent/empty; detail still returns "not_started"

	detail, err := h.Service.GetDetail(r.Context(), offerID, userID)
	if err != nil {
		common.WriteError(w, http.StatusNotFound, "offer not found")
		return
	}
	common.WriteJSON(w, http.StatusOK, detail)
}

func (h *Handler) StartOffer(w http.ResponseWriter, r *http.Request) {
	offerID := chi.URLParam(r, "id")
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	result, err := h.Service.Start(r.Context(), offerID, userID)
	if err != nil {
		common.WriteError(w, http.StatusNotFound, "offer not found")
		return
	}
	common.WriteJSON(w, http.StatusOK, result)
}
