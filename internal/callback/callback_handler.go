package callback

import (
	"log"
	"net/http"
	"strconv"

	"earnsaga-lite/internal/common"
)

type Handler struct {
	Service *Service
}

// HandlePubScale receives PubScale's S2S reward callback:
//   GET /callbacks/pubscale?user_id=...&value=...&token=...&signature=...
//
// Must always return a 2xx once the signature is valid — including on a
// duplicate/replayed token — because PubScale treats any non-2xx as a
// failed delivery and will retry. A duplicate isn't an error condition
// here, it's expected retry behavior we handle via idempotency.
func (h *Handler) HandlePubScale(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	userID := q.Get("user_id")
	valueStr := q.Get("value")
	token := q.Get("token")
	signature := q.Get("signature")

	if userID == "" || valueStr == "" || token == "" || signature == "" {
		common.WriteError(w, http.StatusBadRequest, "missing required callback parameters")
		return
	}

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid value parameter")
		return
	}

	if !h.Service.VerifySignature(userID, value, token, signature) {
		log.Printf("pubscale callback signature mismatch for user_id=%s token=%s", userID, token)
		common.WriteError(w, http.StatusUnauthorized, "invalid signature")
		return
	}

	credited, err := h.Service.HandleCallback(r.Context(), userID, value, token)
	if err != nil {
		log.Printf("pubscale callback processing failed for user_id=%s token=%s: %v", userID, token, err)
		// Still 200 — see idempotency note above. Telling PubScale to retry
		// a callback we may have partially processed risks double-crediting,
		// which is worse than silently accepting and investigating from logs.
		common.WriteJSON(w, http.StatusOK, map[string]string{"status": "acknowledged"})
		return
	}

	status := "credited"
	if !credited {
		status = "duplicate_ignored"
	}
	common.WriteJSON(w, http.StatusOK, map[string]string{"status": status})
}