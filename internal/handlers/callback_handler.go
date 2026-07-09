package handlers

import (
	"net/http"

	"earnsaga-lite/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CallbackHandler struct {
	DB  *pgxpool.Pool
	Cfg *config.Config
}

// PubScaleCallback handles the S2S callback from PubScale.
func (h *CallbackHandler) PubScaleCallback(w http.ResponseWriter, r *http.Request) {
	// TODO: Verify signature using h.Cfg.PubScaleSecretKey
	// TODO: Process the callback (e.g., credit user, update offer status)
	
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
