package handlers

import (
	"net/http"

	"earnsaga-lite/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

type LeaderboardHandler struct {
	DB  *pgxpool.Pool
	Cfg *config.Config
}

type leaderboardEntry struct {
	Name         string  `json:"name"`
	AvatarURL    string  `json:"avatar_url"`
	TotalEarnings float64 `json:"total_earnings"`
}

// GetLeaderboard returns the top users ranked by total earnings.
func (h *LeaderboardHandler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(r.Context(), `
		SELECT u.name, u.avatar_url, SUM(wt.amount) as total_earnings
		FROM users u
		JOIN wallet_transactions wt ON u.id = wt.user_id
		GROUP BY u.id, u.name, u.avatar_url
		ORDER BY total_earnings DESC
		LIMIT 50
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch leaderboard")
		return
	}
	defer rows.Close()

	var entries []leaderboardEntry
	for rows.Next() {
		var e leaderboardEntry
		if err := rows.Scan(&e.Name, &e.AvatarURL, &e.TotalEarnings); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to parse leaderboard data")
			return
		}
		entries = append(entries, e)
	}

	writeJSON(w, http.StatusOK, entries)
}
