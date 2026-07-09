package handlers

import (
	"net/http"

	"earnsaga-lite/internal/auth"
	"earnsaga-lite/internal/services"
)

type UserHandler struct {
	Service *services.UserService
}

func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	profile, err := h.Service.GetProfile(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, profile)
}
