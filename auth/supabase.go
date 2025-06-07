package auth

import (
	"context"

	"github.com/supabase-community/supabase-go"
)

// SupabaseAuth implements AuthProvider using Supabase
type SupabaseAuth struct {
	client *supabase.Client
}

// NewSupabaseAuth creates a new Supabase authentication provider
func NewSupabaseAuth(client *supabase.Client) *SupabaseAuth {
	return &SupabaseAuth{client: client}
}

// SignIn authenticates a user with Supabase
func (s *SupabaseAuth) SignIn(ctx context.Context, username, password string) (string, error) {
	authResponse, err := s.client.Auth.SignInWithEmailPassword(username, password)
	if err != nil {
		return "", err
	}
	return authResponse.AccessToken, nil
}
