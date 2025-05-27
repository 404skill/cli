package auth

import (
	"context"

	"github.com/supabase-community/supabase-go"
)

type AuthProvider interface {
	SignIn(ctx context.Context, username, password string) (string, error)
}

type SupabaseAuth struct {
	client *supabase.Client
}

func NewSupabaseAuth(client *supabase.Client) *SupabaseAuth {
	return &SupabaseAuth{client: client}
}

func (s *SupabaseAuth) SignIn(ctx context.Context, username, password string) (string, error) {
	authResponse, err := s.client.Auth.SignInWithEmailPassword(username, password)
	if err != nil {
		return "", err
	}
	return authResponse.AccessToken, nil
}
