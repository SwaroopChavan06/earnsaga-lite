package handlers

import (
	"net/http"
	"earnsaga-lite/internal/services"
)

type AdminHandler struct {
	Service *services.AdminService
}

func (h *AdminHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	analytics, err := h.Service.GetAnalytics(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch analytics")
		return
	}
	writeJSON(w, http.StatusOK, analytics)
}
