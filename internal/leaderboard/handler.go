package leaderboard

import (
	"net/http"

	"earnsaga-lite/internal/common"
)

type Handler struct {
	Service *Service
}

func (h *Handler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	entries, err := h.Service.GetLeaderboard(r.Context())
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to fetch leaderboard")
		return
	}

	common.WriteJSON(w, http.StatusOK, entries)
}
