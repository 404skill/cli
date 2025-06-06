package login

import (
	"context"
	"fmt"

	"404skill-cli/auth"
	"404skill-cli/config"
)

// AuthService handles authentication business logic
type AuthService struct {
	authProvider  auth.AuthProvider
	configManager *config.ConfigManager
}

// NewAuthService creates a new authentication service
func NewAuthService(authProvider auth.AuthProvider, configManager *config.ConfigManager) *AuthService {
	return &AuthService{
		authProvider:  authProvider,
		configManager: configManager,
	}
}

// LoginResult represents the result of a login attempt
type LoginResult struct {
	Success bool
	Error   string
}

// AttemptLogin performs the complete login flow
func (s *AuthService) AttemptLogin(ctx context.Context, username, password string) LoginResult {
	if username == "" || password == "" {
		return LoginResult{
			Success: false,
			Error:   "Username and password are required",
		}
	}

	// Attempt to sign in
	token, err := s.authProvider.SignIn(ctx, username, password)
	if err != nil {
		return LoginResult{
			Success: false,
			Error:   fmt.Sprintf("Invalid credentials: %v", err),
		}
	}

	// Save configuration using config manager
	if err := s.configManager.UpdateAuthConfig(username, password, token); err != nil {
		return LoginResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to save config: %v", err),
		}
	}

	return LoginResult{
		Success: true,
		Error:   "",
	}
}

// ValidateCredentials performs basic validation on credentials
func (s *AuthService) ValidateCredentials(username, password string) error {
	if username == "" {
		return fmt.Errorf("username is required")
	}
	if password == "" {
		return fmt.Errorf("password is required")
	}
	if len(username) < 2 {
		return fmt.Errorf("username must be at least 2 characters")
	}
	if len(password) < 1 {
		return fmt.Errorf("password cannot be empty")
	}
	return nil
}
