package admin

import (
	"net/http"
	"earnsaga-lite/internal/common"
)

type Handler struct {
	Service *Service
}

func (h *Handler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	analytics, err := h.Service.GetAnalytics(r.Context())
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to fetch analytics")
		return
	}
	common.WriteJSON(w, http.StatusOK, analytics)
}
