package login

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// MockAuthProvider implements auth.AuthProvider for testing
type MockAuthProvider struct {
	signInFunc func(ctx context.Context, username, password string) (string, error)
}

func (m *MockAuthProvider) SignIn(ctx context.Context, username, password string) (string, error) {
	if m.signInFunc != nil {
		return m.signInFunc(ctx, username, password)
	}
	return "mock-token", nil
}

// MockConfigWriter implements auth.ConfigWriter for testing
type MockConfigWriter struct {
	updateAuthConfigFunc func(username, password, accessToken string) error
}

func (m *MockConfigWriter) UpdateAuthConfig(username, password, accessToken string) error {
	if m.updateAuthConfigFunc != nil {
		return m.updateAuthConfigFunc(username, password, accessToken)
	}
	return nil
}

func TestComponent_New(t *testing.T) {
	// Arrange
	mockAuth := &MockAuthProvider{}
	mockConfig := &MockConfigWriter{}

	// Act
	component := New(mockAuth, mockConfig)

	// Assert
	if component == nil {
		t.Error("Expected component to be created")
	}
	if len(component.inputs) != 2 {
		t.Errorf("Expected 2 inputs, got %d", len(component.inputs))
	}
	if component.focusIdx != 0 {
		t.Errorf("Expected focus index to be 0, got %d", component.focusIdx)
	}
	if component.authService == nil {
		t.Error("Expected auth service to be set")
	}
}

func TestComponent_GetUsername(t *testing.T) {
	// Arrange
	mockAuth := &MockAuthProvider{}
	mockConfig := &MockConfigWriter{}
	component := New(mockAuth, mockConfig)

	// Set a test value
	component.inputs[0].SetValue("testuser")

	// Act
	username := component.GetUsername()

	// Assert
	if username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", username)
	}
}

func TestComponent_GetPassword(t *testing.T) {
	// Arrange
	mockAuth := &MockAuthProvider{}
	mockConfig := &MockConfigWriter{}
	component := New(mockAuth, mockConfig)

	// Set a test value
	component.inputs[1].SetValue("testpass")

	// Act
	password := component.GetPassword()

	// Assert
	if password != "testpass" {
		t.Errorf("Expected password 'testpass', got '%s'", password)
	}
}

func TestComponent_SetError(t *testing.T) {
	// Arrange
	mockAuth := &MockAuthProvider{}
	mockConfig := &MockConfigWriter{}
	component := New(mockAuth, mockConfig)

	// Act
	component.SetError("test error")

	// Assert
	if component.errorMsg != "test error" {
		t.Errorf("Expected error 'test error', got '%s'", component.errorMsg)
	}
}

func TestComponent_SetLoggingIn(t *testing.T) {
	// Arrange
	mockAuth := &MockAuthProvider{}
	mockConfig := &MockConfigWriter{}
	component := New(mockAuth, mockConfig)

	// Act
	component.SetLoggingIn(true)

	// Assert
	if !component.loggingIn {
		t.Error("Expected loggingIn to be true")
	}

	// Act
	component.SetLoggingIn(false)

	// Assert
	if component.loggingIn {
		t.Error("Expected loggingIn to be false")
	}
}

func TestComponent_Update_TabNavigation(t *testing.T) {
	// Arrange
	mockAuth := &MockAuthProvider{}
	mockConfig := &MockConfigWriter{}
	component := New(mockAuth, mockConfig)

	// Initially focus should be on first input (index 0)
	if component.focusIdx != 0 {
		t.Errorf("Expected initial focus on input 0, got %d", component.focusIdx)
	}

	// Act - press tab
	updatedComponent, _ := component.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("tab")})

	// Assert - focus should move to password field (index 1)
	if updatedComponent.focusIdx != 1 {
		t.Errorf("Expected focus on input 1 after tab, got %d", updatedComponent.focusIdx)
	}

	// Act - press tab again
	updatedComponent, _ = updatedComponent.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("tab")})

	// Assert - focus should wrap around to username field (index 0)
	if updatedComponent.focusIdx != 0 {
		t.Errorf("Expected focus to wrap to input 0, got %d", updatedComponent.focusIdx)
	}
}

func TestComponent_Update_LoginErrorMsg(t *testing.T) {
	// Arrange
	mockAuth := &MockAuthProvider{}
	mockConfig := &MockConfigWriter{}
	component := New(mockAuth, mockConfig)
	component.loggingIn = true // Set to logging in state

	// Act
	updatedComponent, _ := component.Update(LoginErrorMsg{Error: "test error"})

	// Assert
	if updatedComponent.errorMsg != "test error" {
		t.Errorf("Expected error 'test error', got '%s'", updatedComponent.errorMsg)
	}
	if updatedComponent.loggingIn {
		t.Error("Expected loggingIn to be false after error")
	}
}

func TestComponent_Update_LoginSuccessMsg(t *testing.T) {
	// Arrange
	mockAuth := &MockAuthProvider{}
	mockConfig := &MockConfigWriter{}
	component := New(mockAuth, mockConfig)
	component.loggingIn = true            // Set to logging in state
	component.errorMsg = "previous error" // Set previous error

	// Act
	updatedComponent, _ := component.Update(LoginSuccessMsg{})

	// Assert
	if updatedComponent.errorMsg != "" {
		t.Errorf("Expected error to be cleared, got '%s'", updatedComponent.errorMsg)
	}
	if updatedComponent.loggingIn {
		t.Error("Expected loggingIn to be false after success")
	}
}

func TestComponent_View_ContainsExpectedElements(t *testing.T) {
	// Arrange
	mockAuth := &MockAuthProvider{}
	mockConfig := &MockConfigWriter{}
	component := New(mockAuth, mockConfig)

	// Act
	view := component.View()

	// Assert - focus on functional elements
	expectedElements := []string{
		"Username:",
		"Password:",
		"[tab] switch",   // New footer format
		"[enter] submit", // New footer format
		"[q]",            // Split due to line wrapping
		"quit",           // Split due to line wrapping
	}

	for _, element := range expectedElements {
		if !strings.Contains(view, element) {
			t.Errorf("Expected view to contain '%s', but got: %s", element, view)
		}
	}

	// Verify it's not empty
	if len(view) == 0 {
		t.Error("Expected view to contain content")
	}
}

func TestComponent_View_ShowsError(t *testing.T) {
	// Arrange
	mockAuth := &MockAuthProvider{}
	mockConfig := &MockConfigWriter{}
	component := New(mockAuth, mockConfig)
	component.SetError("Test error message")

	// Act
	view := component.View()

	// Assert
	if !strings.Contains(view, "Test error message") {
		t.Error("Expected view to contain error message")
	}
}

func TestComponent_View_ShowsLoggingIn(t *testing.T) {
	// Arrange
	mockAuth := &MockAuthProvider{}
	mockConfig := &MockConfigWriter{}
	component := New(mockAuth, mockConfig)
	component.SetLoggingIn(true)

	// Act
	view := component.View()

	// Assert
	if !strings.Contains(view, "Logging in...") {
		t.Error("Expected view to contain 'Logging in...' message")
	}
}
