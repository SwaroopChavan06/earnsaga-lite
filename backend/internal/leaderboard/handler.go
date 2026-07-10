package leaderboard

import (
	"fmt"
	"net/http"

	"earnsaga-lite/internal/common"
)

type Handler struct {
	Service     *Service
	Broadcaster *Broadcaster
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
// Server-Sent Events. SSE is one-directional (server → client), which is
// all a leaderboard feed needs — much less connection-management overhead
// than WebSockets.
//
// Instead of each client polling Redis independently, it subscribes to the
// central Broadcaster, which polls once per interval and fans out to every
// connected client. Cost is O(active ranges) in Redis I/O, not O(clients).
//
// Frames are only pushed when the payload changes since last tick, so idle
// clients aren't spammed with duplicate data.
//
// IMPORTANT: this route must be wired WITHOUT the standard 30s timeout
// middleware — a long-lived stream connection would be killed mid-stream.
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

	ch := h.Broadcaster.Subscribe(rangeName)
	defer h.Broadcaster.Unsubscribe(rangeName, ch)

	var lastPayload string

	for {
		select {
		case <-r.Context().Done():
			return
		case data, ok := <-ch:
			if !ok {
				return // broadcaster shut down
			}
			if string(data) == lastPayload {
				continue // no change; don't spam the client
			}
			lastPayload = string(data)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}
