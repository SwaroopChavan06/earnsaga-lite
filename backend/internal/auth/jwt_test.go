package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestIssueAndParseToken_RoundTrip(t *testing.T) {
	secret := "test-secret"
	token, err := IssueToken(secret, "user-1", "user@example.com")
	if err != nil {
		t.Fatalf("unexpected error issuing token: %v", err)
	}

	claims, err := ParseToken(secret, token)
	if err != nil {
		t.Fatalf("unexpected error parsing a freshly issued token: %v", err)
	}
	if claims.UserID != "user-1" || claims.Email != "user@example.com" {
		t.Fatalf("claims did not round-trip, got %+v", claims)
	}
}

func TestParseToken_WrongSecretRejected(t *testing.T) {
	token, err := IssueToken("secret-a", "user-1", "user@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := ParseToken("secret-b", token); err == nil {
		t.Fatal("expected a token signed with a different secret to fail verification")
	}
}

func TestParseToken_GarbageRejected(t *testing.T) {
	if _, err := ParseToken("any-secret", "not-a-jwt-at-all"); err == nil {
		t.Fatal("expected a malformed token string to fail parsing")
	}
}

func TestParseToken_ExpiredRejected(t *testing.T) {
	claims := Claims{
		UserID: "user-1",
		Email:  "user@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("unexpected error signing test token: %v", err)
	}

	if _, err := ParseToken("test-secret", signed); err == nil {
		t.Fatal("expected an expired token to be rejected")
	}
}

func TestParseToken_RejectsAlgNone(t *testing.T) {
	// A token claiming "none" (or any non-HMAC alg) must never be accepted,
	// regardless of secret — this is the classic JWT algorithm-confusion
	// attack ParseToken's explicit method check exists to prevent.
	claims := Claims{UserID: "attacker"}
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	unsigned, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("unexpected error building unsigned token: %v", err)
	}

	if _, err := ParseToken("test-secret", unsigned); err == nil {
		t.Fatal("expected an alg=none token to be rejected")
	}
}
