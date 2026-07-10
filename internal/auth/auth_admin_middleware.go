package auth

import (
	"net/http"

	"earnsaga-lite/internal/common"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RequireAdmin rejects any caller whose user row doesn't have is_admin =
// true. Must sit behind Middleware so a user ID is already in context.
// This was missing after the refactor — the admin route group only
// checked "is this a valid token", not "is this an admin".
func RequireAdmin(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := UserIDFromContext(r.Context())
			if !ok {
				common.WriteError(w, http.StatusUnauthorized, "not authenticated")
				return
			}

			var isAdmin bool
			err := pool.QueryRow(r.Context(), `SELECT is_admin FROM users WHERE id = $1`, userID).Scan(&isAdmin)
			if err != nil || !isAdmin {
				common.WriteError(w, http.StatusForbidden, "admin access required")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
