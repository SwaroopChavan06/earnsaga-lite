package auth

import (
	"context"
	"net/http"
	"strings"

	"earnsaga-lite/internal/common"
)

type contextKey string

const userIDKey contextKey = "userID"
const userEmailKey contextKey = "userEmail"

// Middleware returns a chi-compatible middleware that requires a valid JWT.
// Token resolution order:
//  1. "Authorization: Bearer <token>" header  — used by all regular API calls.
//  2. "?token=<token>" query parameter        — fallback for SSE streams, because
//     the browser's native EventSource API cannot set custom headers.
//
// The query-param fallback is intentionally narrow: it is only reached when
// the Authorization header is absent, so header-based auth is always preferred.
func Middleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var tokenStr string

			header := r.Header.Get("Authorization")
			switch {
			case strings.HasPrefix(header, "Bearer "):
				tokenStr = strings.TrimPrefix(header, "Bearer ")
			case r.URL.Query().Get("token") != "":
				// SSE fallback: EventSource cannot send headers.
				tokenStr = r.URL.Query().Get("token")
			default:
				common.WriteError(w, http.StatusUnauthorized, "missing or malformed authorization header")
				return
			}

			claims, err := ParseToken(secret, tokenStr)
			if err != nil {
				common.WriteError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			ctx = context.WithValue(ctx, userEmailKey, claims.Email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext pulls the authenticated user's ID out of the request
// context. Only call this inside handlers wrapped by Middleware.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok
}

// AdminChecker is the only thing RequireAdmin needs from the user domain —
// kept as a local interface for the same reason as Handler.UserUpserter
// (avoids an auth<->user import cycle). *user.Service satisfies this
// structurally.
type AdminChecker interface {
	IsAdmin(ctx context.Context, userID string) (bool, error)
}

// RequireAdmin rejects any caller whose user isn't flagged is_admin. Must
// sit behind Middleware so a user ID is already in context.
func RequireAdmin(checker AdminChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := UserIDFromContext(r.Context())
			if !ok {
				common.WriteError(w, http.StatusUnauthorized, "not authenticated")
				return
			}

			isAdmin, err := checker.IsAdmin(r.Context(), userID)
			if err != nil || !isAdmin {
				common.WriteError(w, http.StatusForbidden, "admin access required")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
