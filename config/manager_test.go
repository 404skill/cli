package config

import (
	"context"
	"os"
	"testing"
	"time"

	"404skill-cli/auth"
)

// MockAuthService implements AuthService for testing
type MockAuthService struct {
	shouldSucceed bool
	errorMessage  string
}

func (m *MockAuthService) AttemptLogin(ctx context.Context, username, password string) auth.LoginResult {
	if m.shouldSucceed {
		return auth.LoginResult{Success: true, Error: ""}
	}
	return auth.LoginResult{Success: false, Error: m.errorMessage}
}

func newMockAuthService(shouldSucceed bool, errorMessage string) *MockAuthService {
	return &MockAuthService{
		shouldSucceed: shouldSucceed,
		errorMessage:  errorMessage,
	}
}

// Helper function to create a config manager with default mock auth for tests
func newTestConfigManager() *ConfigManager {
	mockAuth := newMockAuthService(true, "")
	return NewConfigManager(mockAuth)
}

// TestNewConfigManager tests the constructor
func TestNewConfigManager(t *testing.T) {
	// Arrange
	mockAuth := newMockAuthService(true, "")

	// Act
	manager := NewConfigManager(mockAuth)

	// Assert
	if manager == nil {
		t.Error("Expected non-nil ConfigManager")
	}
}

// TestConfigManager_HasCredentials_True tests when credentials exist
func TestConfigManager_HasCredentials_True(t *testing.T) {
	// Arrange
	mockAuth := newMockAuthService(true, "")
	manager := NewConfigManager(mockAuth)
	originalPath := ConfigFilePath
	ConfigFilePath = "/tmp/test_has_creds_true.yml"
	defer func() {
		ConfigFilePath = originalPath
		os.Remove("/tmp/test_has_creds_true.yml")
	}()

	cfg := Config{
		Username: "testuser",
		Password: "testpass",
	}
	err := writeConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Act & Assert
	if !manager.HasCredentials() {
		t.Error("Expected HasCredentials to return true when username and password exist")
	}
}

// TestConfigManager_HasCredentials_False_NoConfig tests when config doesn't exist
func TestConfigManager_HasCredentials_False_NoConfig(t *testing.T) {
	// Arrange
	mockAuth := newMockAuthService(true, "")
	manager := NewConfigManager(mockAuth)
	originalPath := ConfigFilePath
	ConfigFilePath = "/tmp/test_has_creds_no_config.yml"
	defer func() {
		ConfigFilePath = originalPath
		os.Remove("/tmp/test_has_creds_no_config.yml")
	}()

	// Act & Assert
	if manager.HasCredentials() {
		t.Error("Expected HasCredentials to return false when config doesn't exist")
	}
}

// TestConfigManager_HasCredentials_False_EmptyUsername tests when username is empty
func TestConfigManager_HasCredentials_False_EmptyUsername(t *testing.T) {
	// Arrange
	mockAuth := newMockAuthService(true, "")
	manager := NewConfigManager(mockAuth)
	originalPath := ConfigFilePath
	ConfigFilePath = "/tmp/test_has_creds_empty_user.yml"
	defer func() {
		ConfigFilePath = originalPath
		os.Remove("/tmp/test_has_creds_empty_user.yml")
	}()

	cfg := Config{
		Username: "",
		Password: "testpass",
	}
	err := writeConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Act & Assert
	if manager.HasCredentials() {
		t.Error("Expected HasCredentials to return false when username is empty")
	}
}

// TestConfigManager_HasCredentials_False_EmptyPassword tests when password is empty
func TestConfigManager_HasCredentials_False_EmptyPassword(t *testing.T) {
	// Arrange
	mockAuth := newMockAuthService(true, "")
	manager := NewConfigManager(mockAuth)
	originalPath := ConfigFilePath
	ConfigFilePath = "/tmp/test_has_creds_empty_pass.yml"
	defer func() {
		ConfigFilePath = originalPath
		os.Remove("/tmp/test_has_creds_empty_pass.yml")
	}()

	cfg := Config{
		Username: "testuser",
		Password: "",
	}
	err := writeConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Act & Assert
	if manager.HasCredentials() {
		t.Error("Expected HasCredentials to return false when password is empty")
	}
}

// TestConfigManager_GetDownloadedProjects_WithProjects tests when downloaded projects exist
func TestConfigManager_GetDownloadedProjects_WithProjects(t *testing.T) {
	// Arrange
	manager := newTestConfigManager()
	originalPath := ConfigFilePath
	ConfigFilePath = "/tmp/test_get_downloaded_with.yml"
	defer func() {
		ConfigFilePath = originalPath
		os.Remove("/tmp/test_get_downloaded_with.yml")
	}()

	expectedProjects := map[string]bool{
		"project1": true,
		"project2": true,
		"project3": false, // false values should also be preserved
	}
	cfg := Config{
		DownloadedProjects: expectedProjects,
	}
	err := writeConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Act
	projects := manager.GetDownloadedProjects()

	// Assert
	if len(projects) != len(expectedProjects) {
		t.Errorf("Expected %d projects, got %d", len(expectedProjects), len(projects))
	}
	for projectID, expected := range expectedProjects {
		if projects[projectID] != expected {
			t.Errorf("Expected project %s to be %v, got %v", projectID, expected, projects[projectID])
		}
	}
}

// TestConfigManager_GetDownloadedProjects_NoConfig tests when config doesn't exist
func TestConfigManager_GetDownloadedProjects_NoConfig(t *testing.T) {
	// Arrange
	manager := newTestConfigManager()
	originalPath := ConfigFilePath
	ConfigFilePath = "/tmp/test_get_downloaded_no_config.yml"
	defer func() {
		ConfigFilePath = originalPath
		os.Remove("/tmp/test_get_downloaded_no_config.yml")
	}()

	// Act
	projects := manager.GetDownloadedProjects()

	// Assert
	if projects == nil {
		t.Error("Expected non-nil map")
	}
	if len(projects) != 0 {
		t.Errorf("Expected empty map, got %d items", len(projects))
	}
}

// TestConfigManager_GetDownloadedProjects_NilMap tests when DownloadedProjects is nil
func TestConfigManager_GetDownloadedProjects_NilMap(t *testing.T) {
	// Arrange
	manager := newTestConfigManager()
	originalPath := ConfigFilePath
	ConfigFilePath = "/tmp/test_get_downloaded_nil.yml"
	defer func() {
		ConfigFilePath = originalPath
		os.Remove("/tmp/test_get_downloaded_nil.yml")
	}()

	cfg := Config{
		Username:           "testuser",
		DownloadedProjects: nil,
	}
	err := writeConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Act
	projects := manager.GetDownloadedProjects()

	// Assert
	if projects == nil {
		t.Error("Expected non-nil map")
	}
	if len(projects) != 0 {
		t.Errorf("Expected empty map, got %d items", len(projects))
	}
}

func TestConfigManager_UpdateAuthConfig_NewConfig(t *testing.T) {
	// Arrange
	manager := newTestConfigManager()

	// Create a temporary config file path for testing
	originalPath := ConfigFilePath
	ConfigFilePath = "/tmp/test_config_new.yml"
	defer func() {
		ConfigFilePath = originalPath
		os.Remove("/tmp/test_config_new.yml")
	}()

	// Act
	err := manager.UpdateAuthConfig("testuser", "testpass", "test-token")

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify the config was saved correctly
	cfg, err := readConfig()
	if err != nil {
		t.Errorf("Failed to read config: %v", err)
	}

	if cfg.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", cfg.Username)
	}
	if cfg.Password != "testpass" {
		t.Errorf("Expected password 'testpass', got '%s'", cfg.Password)
	}
	if cfg.AccessToken != "test-token" {
		t.Errorf("Expected token 'test-token', got '%s'", cfg.AccessToken)
	}
	if cfg.DownloadedProjects == nil {
		t.Error("Expected DownloadedProjects to be initialized")
	}
	if time.Since(cfg.LastUpdated) > time.Minute {
		t.Error("Expected LastUpdated to be recent")
	}
}

func TestConfigManager_UpdateAuthConfig_PreservesExistingData(t *testing.T) {
	// Arrange
	manager := newTestConfigManager()

	// Create a temporary config file path for testing
	originalPath := ConfigFilePath
	ConfigFilePath = "/tmp/test_config_preserve.yml"
	defer func() {
		ConfigFilePath = originalPath
		os.Remove("/tmp/test_config_preserve.yml")
	}()

	// First, create a config with downloaded projects
	initialCfg := Config{
		Username:           "olduser",
		Password:           "oldpass",
		AccessToken:        "old-token",
		LastUpdated:        time.Now().Add(-time.Hour),
		DownloadedProjects: map[string]bool{"project1": true, "project2": true},
	}
	err := writeConfig(initialCfg)
	if err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	// Act
	err = manager.UpdateAuthConfig("newuser", "newpass", "new-token")

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify the config was updated correctly
	cfg, err := readConfig()
	if err != nil {
		t.Errorf("Failed to read config: %v", err)
	}

	// Check that auth fields are updated
	if cfg.Username != "newuser" {
		t.Errorf("Expected username 'newuser', got '%s'", cfg.Username)
	}
	if cfg.Password != "newpass" {
		t.Errorf("Expected password 'newpass', got '%s'", cfg.Password)
	}
	if cfg.AccessToken != "new-token" {
		t.Errorf("Expected token 'new-token', got '%s'", cfg.AccessToken)
	}

	// Check that existing downloaded projects are preserved
	if !cfg.DownloadedProjects["project1"] {
		t.Error("Expected project1 to be preserved")
	}
	if !cfg.DownloadedProjects["project2"] {
		t.Error("Expected project2 to be preserved")
	}

	// Check that LastUpdated is recent
	if time.Since(cfg.LastUpdated) > time.Minute {
		t.Error("Expected LastUpdated to be recent")
	}
}

func TestConfigManager_IsProjectDownloaded(t *testing.T) {
	// Arrange
	manager := newTestConfigManager()

	// Create a temporary config file path for testing
	originalPath := ConfigFilePath
	ConfigFilePath = "/tmp/test_config_project.yml"
	defer func() {
		ConfigFilePath = originalPath
		os.Remove("/tmp/test_config_project.yml")
	}()

	// Create a config with some downloaded projects
	cfg := Config{
		DownloadedProjects: map[string]bool{"project1": true},
	}
	err := writeConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Act & Assert
	if !manager.IsProjectDownloaded("project1") {
		t.Error("Expected project1 to be downloaded")
	}
	if manager.IsProjectDownloaded("project2") {
		t.Error("Expected project2 to not be downloaded")
	}
}

// TestConfigManager_IsProjectDownloaded_NoConfig tests when config doesn't exist
func TestConfigManager_IsProjectDownloaded_NoConfig(t *testing.T) {
	// Arrange
	manager := newTestConfigManager()
	originalPath := ConfigFilePath
	ConfigFilePath = "/tmp/test_is_downloaded_no_config.yml"
	defer func() {
		ConfigFilePath = originalPath
		os.Remove("/tmp/test_is_downloaded_no_config.yml")
	}()

	// Act & Assert
	if manager.IsProjectDownloaded("project1") {
		t.Error("Expected project1 to not be downloaded when config doesn't exist")
	}
}

// TestConfigManager_IsProjectDownloaded_NilMap tests when DownloadedProjects is nil
func TestConfigManager_IsProjectDownloaded_NilMap(t *testing.T) {
	// Arrange
	manager := newTestConfigManager()
	originalPath := ConfigFilePath
	ConfigFilePath = "/tmp/test_is_downloaded_nil.yml"
	defer func() {
		ConfigFilePath = originalPath
		os.Remove("/tmp/test_is_downloaded_nil.yml")
	}()

	cfg := Config{
		Username:           "testuser",
		DownloadedProjects: nil,
	}
	err := writeConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Act & Assert
	if manager.IsProjectDownloaded("project1") {
		t.Error("Expected project1 to not be downloaded when DownloadedProjects is nil")
	}
}

func TestConfigManager_UpdateDownloadedProject(t *testing.T) {
	// Arrange
	manager := newTestConfigManager()

	// Create a temporary config file path for testing
	originalPath := ConfigFilePath
	ConfigFilePath = "/tmp/test_config_download.yml"
	defer func() {
		ConfigFilePath = originalPath
		os.Remove("/tmp/test_config_download.yml")
	}()

	// Create an initial config
	cfg := Config{
		Username:           "testuser",
		DownloadedProjects: map[string]bool{"project1": true},
	}
	err := writeConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	// Act
	err = manager.UpdateDownloadedProject("project2")

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify the project was added
	updatedCfg, err := readConfig()
	if err != nil {
		t.Errorf("Failed to read updated config: %v", err)
	}

	if !updatedCfg.DownloadedProjects["project1"] {
		t.Error("Expected existing project1 to be preserved")
	}
	if !updatedCfg.DownloadedProjects["project2"] {
		t.Error("Expected project2 to be added")
	}
	if updatedCfg.Username != "testuser" {
		t.Error("Expected other config fields to be preserved")
	}
}

// TestConfigManager_UpdateDownloadedProject_NoConfig tests when config doesn't exist
func TestConfigManager_UpdateDownloadedProject_NoConfig(t *testing.T) {
	// Arrange
	manager := newTestConfigManager()
	originalPath := ConfigFilePath
	ConfigFilePath = "/tmp/test_update_downloaded_no_config.yml"
	defer func() {
		ConfigFilePath = originalPath
		os.Remove("/tmp/test_update_downloaded_no_config.yml")
	}()

	// Act
	err := manager.UpdateDownloadedProject("project1")

	// Assert
	if err == nil {
		t.Error("Expected error when config doesn't exist")
	}
}

// TestConfigManager_UpdateDownloadedProject_NilMap tests when DownloadedProjects is nil
func TestConfigManager_UpdateDownloadedProject_NilMap(t *testing.T) {
	// Arrange
	manager := newTestConfigManager()
	originalPath := ConfigFilePath
	ConfigFilePath = "/tmp/test_update_downloaded_nil.yml"
	defer func() {
		ConfigFilePath = originalPath
		os.Remove("/tmp/test_update_downloaded_nil.yml")
	}()

	cfg := Config{
		Username:           "testuser",
		DownloadedProjects: nil,
	}
	err := writeConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Act
	err = manager.UpdateDownloadedProject("project1")

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify the project was added and map was initialized
	updatedCfg, err := readConfig()
	if err != nil {
		t.Errorf("Failed to read updated config: %v", err)
	}

	if updatedCfg.DownloadedProjects == nil {
		t.Error("Expected DownloadedProjects to be initialized")
	}
	if !updatedCfg.DownloadedProjects["project1"] {
		t.Error("Expected project1 to be added")
	}
	if updatedCfg.Username != "testuser" {
		t.Error("Expected other config fields to be preserved")
	}
}

// TestConfigManager_GetToken_ValidToken tests when token exists and is not expired
func TestConfigManager_GetToken_ValidToken(t *testing.T) {
	// Arrange
	manager := newTestConfigManager()
	originalPath := ConfigFilePath
	ConfigFilePath = "/tmp/test_get_token_valid.yml"
	defer func() {
		ConfigFilePath = originalPath
		os.Remove("/tmp/test_get_token_valid.yml")
	}()

	cfg := Config{
		Username:    "testuser",
		Password:    "testpass",
		AccessToken: "valid-token",
		LastUpdated: time.Now().Add(-time.Hour), // Not expired (less than 24 hours)
	}
	err := writeConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Act
	token, err := manager.GetToken()

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if token != "valid-token" {
		t.Errorf("Expected token 'valid-token', got '%s'", token)
	}
}

// TestConfigManager_GetToken_NoConfig tests when config doesn't exist
func TestConfigManager_GetToken_NoConfig(t *testing.T) {
	// Arrange
	manager := newTestConfigManager()
	originalPath := ConfigFilePath
	ConfigFilePath = "/tmp/test_get_token_no_config.yml"
	defer func() {
		ConfigFilePath = originalPath
		os.Remove("/tmp/test_get_token_no_config.yml")
	}()

	// Act
	_, err := manager.GetToken()

	// Assert
	if err == nil {
		t.Error("Expected error when config doesn't exist")
	}
}

// TestConfigManager_GetToken_EmptyToken tests when token is empty
func TestConfigManager_GetToken_EmptyToken(t *testing.T) {
	// Arrange
	mockAuth := newMockAuthService(false, "auth failed")
	manager := NewConfigManager(mockAuth)
	originalPath := ConfigFilePath
	ConfigFilePath = "/tmp/test_get_token_empty.yml"
	defer func() {
		ConfigFilePath = originalPath
		os.Remove("/tmp/test_get_token_empty.yml")
	}()

	cfg := Config{
		Username:    "testuser",
		Password:    "testpass",
		AccessToken: "", // Empty token
		LastUpdated: time.Now(),
	}
	err := writeConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Act
	_, err = manager.GetToken()

	// Assert - This will fail because it tries to create supabase client
	// But that's expected behavior - we need credentials to refresh
	if err == nil {
		t.Error("Expected error when trying to refresh with empty token")
	}
}

// TestConfigManager_GetToken_ExpiredToken tests when token is expired
func TestConfigManager_GetToken_ExpiredToken(t *testing.T) {
	// Arrange
	mockAuth := newMockAuthService(false, "auth failed")
	manager := NewConfigManager(mockAuth)
	originalPath := ConfigFilePath
	ConfigFilePath = "/tmp/test_get_token_expired.yml"
	defer func() {
		ConfigFilePath = originalPath
		os.Remove("/tmp/test_get_token_expired.yml")
	}()

	cfg := Config{
		Username:    "testuser",
		Password:    "testpass",
		AccessToken: "expired-token",
		LastUpdated: time.Now().Add(-25 * time.Hour), // Expired (more than 24 hours)
	}
	err := writeConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Act
	_, err = manager.GetToken()

	// Assert - This will fail because it tries to create supabase client
	// But that's expected behavior - we need valid credentials to refresh
	if err == nil {
		t.Error("Expected error when trying to refresh expired token")
	}
}
