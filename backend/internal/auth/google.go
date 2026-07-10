package auth

import (
	"context"
	"fmt"

	"google.golang.org/api/idtoken"
)

type GoogleUser struct {
	Sub     string
	Email   string
	Name    string
	Picture string
}

// VerifyGoogleIDToken validates the id_token the frontend receives from
// Google Sign-In and pulls out the profile fields we need. This is the
// server-side check that stops anyone from just POSTing a fake user.
func VerifyGoogleIDToken(ctx context.Context, clientID, rawToken string) (*GoogleUser, error) {
	payload, err := idtoken.Validate(ctx, rawToken, clientID)
	if err != nil {
		return nil, fmt.Errorf("invalid google id token: %w", err)
	}

	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)
	picture, _ := payload.Claims["picture"].(string)

	if email == "" {
		return nil, fmt.Errorf("google token missing email claim")
	}

	return &GoogleUser{
		Sub:     payload.Subject,
		Email:   email,
		Name:    name,
		Picture: picture,
	}, nil
}
