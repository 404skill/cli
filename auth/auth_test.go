package auth

import (
	"context"
	"errors"
	"testing"
)

// MockAuthProvider implements AuthProvider for testing
type MockAuthProvider struct {
	signInFunc func(ctx context.Context, username, password string) (string, error)
}

func (m *MockAuthProvider) SignIn(ctx context.Context, username, password string) (string, error) {
	if m.signInFunc != nil {
		return m.signInFunc(ctx, username, password)
	}
	return "mock-token", nil
}

// MockConfigWriter implements ConfigWriter for testing
type MockConfigWriter struct {
	updateAuthConfigFunc func(username, password, accessToken string) error
}

func (m *MockConfigWriter) UpdateAuthConfig(username, password, accessToken string) error {
	if m.updateAuthConfigFunc != nil {
		return m.updateAuthConfigFunc(username, password, accessToken)
	}
	return nil
}

func TestAuthService_AttemptLogin_Success(t *testing.T) {
	// Arrange
	mockAuth := &MockAuthProvider{
		signInFunc: func(ctx context.Context, username, password string) (string, error) {
			return "test-token", nil
		},
	}
	mockConfig := &MockConfigWriter{}
	service := NewAuthService(mockAuth, mockConfig)

	// Act
	result := service.AttemptLogin(context.Background(), "testuser", "testpass")

	// Assert
	if !result.Success {
		t.Errorf("Expected login to succeed, but got error: %s", result.Error)
	}
	if result.Error != "" {
		t.Errorf("Expected no error, but got: %s", result.Error)
	}
}

func TestAuthService_AttemptLogin_InvalidCredentials(t *testing.T) {
	// Arrange
	mockAuth := &MockAuthProvider{
		signInFunc: func(ctx context.Context, username, password string) (string, error) {
			return "", errors.New("invalid credentials")
		},
	}
	mockConfig := &MockConfigWriter{}
	service := NewAuthService(mockAuth, mockConfig)

	// Act
	result := service.AttemptLogin(context.Background(), "wronguser", "wrongpass")

	// Assert
	if result.Success {
		t.Error("Expected login to fail, but it succeeded")
	}
	expectedError := "Invalid credentials: invalid credentials"
	if result.Error != expectedError {
		t.Errorf("Expected error '%s', but got '%s'", expectedError, result.Error)
	}
}

func TestAuthService_AttemptLogin_EmptyUsername(t *testing.T) {
	// Arrange
	mockAuth := &MockAuthProvider{}
	mockConfig := &MockConfigWriter{}
	service := NewAuthService(mockAuth, mockConfig)

	// Act
	result := service.AttemptLogin(context.Background(), "", "password")

	// Assert
	if result.Success {
		t.Error("Expected login to fail with empty username")
	}
	expectedError := "Username and password are required"
	if result.Error != expectedError {
		t.Errorf("Expected error '%s', but got '%s'", expectedError, result.Error)
	}
}

func TestAuthService_AttemptLogin_EmptyPassword(t *testing.T) {
	// Arrange
	mockAuth := &MockAuthProvider{}
	mockConfig := &MockConfigWriter{}
	service := NewAuthService(mockAuth, mockConfig)

	// Act
	result := service.AttemptLogin(context.Background(), "username", "")

	// Assert
	if result.Success {
		t.Error("Expected login to fail with empty password")
	}
	expectedError := "Username and password are required"
	if result.Error != expectedError {
		t.Errorf("Expected error '%s', but got '%s'", expectedError, result.Error)
	}
}

func TestAuthService_AttemptLogin_ConfigSaveError(t *testing.T) {
	// Arrange
	mockAuth := &MockAuthProvider{
		signInFunc: func(ctx context.Context, username, password string) (string, error) {
			return "test-token", nil
		},
	}
	mockConfig := &MockConfigWriter{
		updateAuthConfigFunc: func(username, password, accessToken string) error {
			return errors.New("config save failed")
		},
	}
	service := NewAuthService(mockAuth, mockConfig)

	// Act
	result := service.AttemptLogin(context.Background(), "testuser", "testpass")

	// Assert
	if result.Success {
		t.Error("Expected login to fail when config save fails")
	}
	expectedError := "Failed to save config: config save failed"
	if result.Error != expectedError {
		t.Errorf("Expected error '%s', but got '%s'", expectedError, result.Error)
	}
}

func TestAuthService_ValidateCredentials_Valid(t *testing.T) {
	// Arrange
	service := &AuthService{}

	// Act
	err := service.ValidateCredentials("testuser", "testpass")

	// Assert
	if err != nil {
		t.Errorf("Expected no error for valid credentials, but got: %s", err.Error())
	}
}

func TestAuthService_ValidateCredentials_EmptyUsername(t *testing.T) {
	// Arrange
	service := &AuthService{}

	// Act
	err := service.ValidateCredentials("", "testpass")

	// Assert
	if err == nil {
		t.Error("Expected error for empty username")
	}
	expectedError := "username is required"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', but got '%s'", expectedError, err.Error())
	}
}

func TestAuthService_ValidateCredentials_EmptyPassword(t *testing.T) {
	// Arrange
	service := &AuthService{}

	// Act
	err := service.ValidateCredentials("testuser", "")

	// Assert
	if err == nil {
		t.Error("Expected error for empty password")
	}
	expectedError := "password is required"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', but got '%s'", expectedError, err.Error())
	}
}

func TestAuthService_ValidateCredentials_ShortUsername(t *testing.T) {
	// Arrange
	service := &AuthService{}

	// Act
	err := service.ValidateCredentials("a", "testpass")

	// Assert
	if err == nil {
		t.Error("Expected error for short username")
	}
	expectedError := "username must be at least 2 characters"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', but got '%s'", expectedError, err.Error())
	}
}

func TestNewAuthService(t *testing.T) {
	// Arrange
	mockAuth := &MockAuthProvider{}
	mockConfig := &MockConfigWriter{}

	// Act
	service := NewAuthService(mockAuth, mockConfig)

	// Assert
	if service == nil {
		t.Error("Expected service to be created")
	}
	if service.authProvider != mockAuth {
		t.Error("Expected auth provider to be set correctly")
	}
	if service.configWriter != mockConfig {
		t.Error("Expected config writer to be set correctly")
	}
}
