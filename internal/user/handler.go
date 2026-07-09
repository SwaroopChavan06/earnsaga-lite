package user

import (
	"net/http"

	"earnsaga-lite/internal/auth"
	"earnsaga-lite/internal/common"
)

type Handler struct {
	Service *Service
}

func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	profile, err := h.Service.GetProfile(r.Context(), userID)
	if err != nil {
		common.WriteError(w, http.StatusNotFound, "user not found")
		return
	}

	common.WriteJSON(w, http.StatusOK, profile)
}
