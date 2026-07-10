package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func passThroughHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestMiddleware_RejectsMissingHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()

	Middleware("secret")(passThroughHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with no Authorization header, got %d", rec.Code)
	}
}

func TestMiddleware_RejectsMalformedHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "NotBearer sometoken")
	rec := httptest.NewRecorder()

	Middleware("secret")(passThroughHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for a header without the Bearer prefix, got %d", rec.Code)
	}
}

func TestMiddleware_RejectsInvalidToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer garbage-not-a-real-jwt")
	rec := httptest.NewRecorder()

	Middleware("secret")(passThroughHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for an invalid token, got %d", rec.Code)
	}
}

func TestMiddleware_AcceptsValidTokenAndInjectsUserID(t *testing.T) {
	secret := "secret"
	token, err := IssueToken(secret, "user-1", "user@example.com")
	if err != nil {
		t.Fatalf("unexpected error issuing token: %v", err)
	}

	var sawUserID string
	var sawOK bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawUserID, sawOK = UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	Middleware(secret)(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for a valid token, got %d", rec.Code)
	}
	if !sawOK || sawUserID != "user-1" {
		t.Fatalf("expected downstream handler to see user-1 in context, got ok=%v id=%q", sawOK, sawUserID)
	}
}

type fakeAdminChecker struct {
	isAdmin bool
	err     error
}

func (f fakeAdminChecker) IsAdmin(ctx context.Context, userID string) (bool, error) {
	return f.isAdmin, f.err
}

func TestRequireAdmin_RejectsUnauthenticated(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/admin/analytics", nil)
	rec := httptest.NewRecorder()

	RequireAdmin(fakeAdminChecker{isAdmin: true})(passThroughHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with no user id in context, got %d", rec.Code)
	}
}

func TestRequireAdmin_RejectsNonAdmin(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/admin/analytics", nil)
	req = req.WithContext(context.WithValue(req.Context(), userIDKey, "user-1"))
	rec := httptest.NewRecorder()

	RequireAdmin(fakeAdminChecker{isAdmin: false})(passThroughHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for a non-admin user, got %d", rec.Code)
	}
}

func TestRequireAdmin_RejectsOnCheckerError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/admin/analytics", nil)
	req = req.WithContext(context.WithValue(req.Context(), userIDKey, "user-1"))
	rec := httptest.NewRecorder()

	RequireAdmin(fakeAdminChecker{err: context.DeadlineExceeded})(passThroughHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected a checker error to fail closed (403), got %d", rec.Code)
	}
}

func TestRequireAdmin_AllowsAdmin(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/admin/analytics", nil)
	req = req.WithContext(context.WithValue(req.Context(), userIDKey, "admin-1"))
	rec := httptest.NewRecorder()

	RequireAdmin(fakeAdminChecker{isAdmin: true})(passThroughHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for an actual admin, got %d", rec.Code)
	}
}
