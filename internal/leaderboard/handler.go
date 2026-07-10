package leaderboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"earnsaga-lite/internal/common"
)

type Handler struct {
	Service *Service
}

// GetLeaderboard handles GET /leaderboard?range=daily|weekly|alltime
func (h *Handler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	rangeName := r.URL.Query().Get("range")
	entries, err := h.Service.GetLeaderboard(r.Context(), rangeName)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to fetch leaderboard")
		return
	}
	common.WriteJSON(w, http.StatusOK, entries)
}

// StreamLeaderboard handles GET /leaderboard/stream?range=daily via
// Server-Sent Events. SSE is one-directional (server -> client), which is
// all a leaderboard feed needs — much less connection-management overhead
// than WebSockets. Polls Redis every 3s and only pushes a frame when the
// ranked list actually changed, so idle clients aren't spammed.
//
// IMPORTANT: this route must be wired in main.go WITHOUT the standard 30s
// request timeout middleware, or the connection gets killed mid-stream.
func (h *Handler) StreamLeaderboard(w http.ResponseWriter, r *http.Request) {
	rangeName := r.URL.Query().Get("range")

	flusher, ok := w.(http.Flusher)
	if !ok {
		common.WriteError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	var lastPayload string

	push := func() {
		entries, err := h.Service.GetLeaderboard(r.Context(), rangeName)
		if err != nil {
			return // skip this tick on a transient error, don't kill the whole stream
		}
		data, err := json.Marshal(entries)
		if err != nil {
			return
		}
		if string(data) == lastPayload {
			return // no change since last tick, don't spam the client
		}
		lastPayload = string(data)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	push() // send current state immediately on connect

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			push()
		}
	}
}
