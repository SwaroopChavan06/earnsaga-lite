package handlers

import (
	"net/http"

	"earnsaga-lite/internal/services"
)

type LeaderboardHandler struct {
	Service *services.LeaderboardService
}

func (h *LeaderboardHandler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	entries, err := h.Service.GetLeaderboard(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch leaderboard")
		return
	}

	writeJSON(w, http.StatusOK, entries)
}
