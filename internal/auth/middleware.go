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

// Middleware returns a chi-compatible middleware that requires a valid
// "Authorization: Bearer <token>" header, and injects the user ID into
// the request context for downstream handlers.
func Middleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" || !strings.HasPrefix(header, "Bearer ") {
				common.WriteError(w, http.StatusUnauthorized, "missing or malformed authorization header")
				return
			}

			tokenStr := strings.TrimPrefix(header, "Bearer ")
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
