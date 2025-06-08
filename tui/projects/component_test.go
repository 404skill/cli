package projects

import (
	"404skill-cli/api"
	"404skill-cli/auth"
	"404skill-cli/config"
	"404skill-cli/filesystem"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// MockAuthService implements config.AuthService for testing
type MockAuthService struct{}

func (m *MockAuthService) AttemptLogin(ctx context.Context, username, password string) auth.LoginResult {
	return auth.LoginResult{Success: true, Error: ""}
}

// Helper function to create a config manager with mock auth for tests
func newTestConfigManager() *config.ConfigManager {
	mockAuth := &MockAuthService{}
	return config.NewConfigManager(mockAuth)
}

// MockClient implements api.ClientInterface for testing
type MockClient struct {
	listProjectsFunc      func(ctx context.Context) ([]api.Project, error)
	initProjectFunc       func(ctx context.Context, projectID string) error
	bulkUpdateProfileFunc func(ctx context.Context, failed, passed []string, projectID string) error
}

func (m *MockClient) ListProjects(ctx context.Context) ([]api.Project, error) {
	if m.listProjectsFunc != nil {
		return m.listProjectsFunc(ctx)
	}
	return []api.Project{}, nil
}

func (m *MockClient) InitializeProject(ctx context.Context, projectID string) error {
	if m.initProjectFunc != nil {
		return m.initProjectFunc(ctx, projectID)
	}
	return nil
}

func (m *MockClient) BulkUpdateProfileTests(ctx context.Context, failed, passed []string, projectID string) error {
	if m.bulkUpdateProfileFunc != nil {
		return m.bulkUpdateProfileFunc(ctx, failed, passed, projectID)
	}
	return nil
}

// setupIsolatedConfig creates a unique config file for testing with an initial config
func setupIsolatedConfig(t *testing.T) (*config.ConfigManager, func()) {
	t.Helper()

	originalPath := config.ConfigFilePath
	testConfigPath := fmt.Sprintf("/tmp/test_projects_%s.yml", t.Name())
	config.ConfigFilePath = testConfigPath

	// Create an initial config file using the public API
	manager := newTestConfigManager()
	err := manager.UpdateAuthConfig("testuser", "testpass", "test-token")
	if err != nil {
		t.Fatalf("Failed to create initial test config: %v", err)
	}

	cleanup := func() {
		config.ConfigFilePath = originalPath
		os.Remove(testConfigPath)
	}

	return manager, cleanup
}

func TestComponent_New(t *testing.T) {
	// Arrange
	mockClient := &MockClient{}
	configManager := newTestConfigManager()
	fileManager := filesystem.NewManager()

	// Act
	component := New(mockClient, configManager, fileManager)

	// Assert
	if component == nil {
		t.Error("Expected component to be created")
	}
	if component.client != mockClient {
		t.Error("Expected client to be set")
	}
	if component.configManager != configManager {
		t.Error("Expected config manager to be set")
	}
	if component.fileManager != fileManager {
		t.Error("Expected file manager to be set")
	}
	if component.table == nil {
		t.Error("Expected table component to be created")
	}
	if component.loading {
		t.Error("Expected loading to be false initially")
	}
}

func TestComponent_GetProjectStatus_Downloaded(t *testing.T) {
	// Arrange
	mockClient := &MockClient{}
	configManager, cleanup := setupIsolatedConfig(t)
	defer cleanup()
	fileManager := filesystem.NewManager()
	component := New(mockClient, configManager, fileManager)

	// Mark project as downloaded
	err := configManager.UpdateDownloadedProject("test-project")
	if err != nil {
		t.Fatalf("Failed to mark project as downloaded: %v", err)
	}

	// Act
	status := component.GetProjectStatus("test-project")

	// Assert - Project should show as downloaded
	if status != "✓ Downloaded" {
		t.Errorf("Expected '✓ Downloaded', got '%s'", status)
	}
}

func TestComponent_GetProjectStatus_NotDownloaded(t *testing.T) {
	// Arrange
	mockClient := &MockClient{}
	configManager := newTestConfigManager()
	fileManager := filesystem.NewManager()
	component := New(mockClient, configManager, fileManager)

	// Act
	status := component.GetProjectStatus("not-downloaded-project")

	// Assert
	if status != "" {
		t.Errorf("Expected empty string, got '%s'", status)
	}
}

func TestComponent_SetLoading(t *testing.T) {
	// Arrange
	mockClient := &MockClient{}
	configManager := newTestConfigManager()
	fileManager := filesystem.NewManager()
	component := New(mockClient, configManager, fileManager)

	// Act
	component.SetLoading(true)

	// Assert
	if !component.loading {
		t.Error("Expected loading to be true")
	}

	// Act
	component.SetLoading(false)

	// Assert
	if component.loading {
		t.Error("Expected loading to be false")
	}
}

func TestComponent_SetProjects(t *testing.T) {
	// Arrange
	mockClient := &MockClient{}
	configManager := newTestConfigManager()
	fileManager := filesystem.NewManager()
	component := New(mockClient, configManager, fileManager)

	projects := []api.Project{
		{ID: "1", Name: "Project 1", Language: "Go", Difficulty: "Easy", EstimatedDurationInMinutes: 30},
		{ID: "2", Name: "Project 2", Language: "Python", Difficulty: "Medium", EstimatedDurationInMinutes: 60},
	}

	// Act
	component.SetProjects(projects)

	// Assert
	if len(component.projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(component.projects))
	}
	if component.projects[0].Name != "Project 1" {
		t.Errorf("Expected first project name 'Project 1', got '%s'", component.projects[0].Name)
	}
	if component.loading {
		t.Error("Expected loading to be false after setting projects")
	}
}

func TestComponent_SetError(t *testing.T) {
	// Arrange
	mockClient := &MockClient{}
	configManager := newTestConfigManager()
	fileManager := filesystem.NewManager()
	component := New(mockClient, configManager, fileManager)

	// Act
	component.SetError("test error")

	// Assert
	if component.errorMsg != "test error" {
		t.Errorf("Expected error 'test error', got '%s'", component.errorMsg)
	}
	if component.loading {
		t.Error("Expected loading to be false after error")
	}
}

func TestComponent_Update_ProjectsList(t *testing.T) {
	// Arrange
	mockClient := &MockClient{}
	configManager := newTestConfigManager()
	fileManager := filesystem.NewManager()
	component := New(mockClient, configManager, fileManager)

	projects := []api.Project{
		{ID: "1", Name: "Test Project", Language: "Go", Difficulty: "Easy", EstimatedDurationInMinutes: 30},
	}

	// Act
	updatedComponent, _ := component.Update(projects)

	// Assert
	if len(updatedComponent.projects) != 1 {
		t.Errorf("Expected 1 project, got %d", len(updatedComponent.projects))
	}
	if updatedComponent.projects[0].Name != "Test Project" {
		t.Errorf("Expected project name 'Test Project', got '%s'", updatedComponent.projects[0].Name)
	}
}

func TestComponent_Update_ProjectsError(t *testing.T) {
	// Arrange
	mockClient := &MockClient{}
	configManager := newTestConfigManager()
	fileManager := filesystem.NewManager()
	component := New(mockClient, configManager, fileManager)

	// Act
	updatedComponent, _ := component.Update(ProjectsErrorMsg{Error: "test error"})

	// Assert
	if updatedComponent.errorMsg != "test error" {
		t.Errorf("Expected error 'test error', got '%s'", updatedComponent.errorMsg)
	}
}

func TestComponent_Update_EnterKey_NotDownloaded(t *testing.T) {
	// Arrange
	mockClient := &MockClient{}
	configManager, cleanup := setupIsolatedConfig(t)
	defer cleanup()
	fileManager := filesystem.NewManager()
	component := New(mockClient, configManager, fileManager)

	// Set up a project in the table (ensure it's not downloaded)
	projects := []api.Project{
		{ID: "test-project", Name: "Test Project", Language: "Go", Difficulty: "Easy", EstimatedDurationInMinutes: 30},
	}
	component.SetProjects(projects)

	// Act
	updatedComponent, cmd := component.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Assert
	if updatedComponent == nil {
		t.Error("Expected component to be returned")
	}

	// Command might be nil if no project is highlighted (depends on table state)
	if cmd != nil {
		// If there's a command, execute it to check the message type
		msg := cmd()
		if _, ok := msg.(ProjectSelectedMsg); !ok {
			t.Errorf("Expected ProjectSelectedMsg, got %T", msg)
		}
	}
}

func TestComponent_Update_EnterKey_Downloaded_FileNotFound(t *testing.T) {
	// Arrange
	mockClient := &MockClient{}
	configManager, cleanup := setupIsolatedConfig(t)
	defer cleanup()
	fileManager := filesystem.NewManager()
	component := New(mockClient, configManager, fileManager)

	// Set up a project in the table and mark it as downloaded
	projects := []api.Project{
		{ID: "test-project", Name: "Test Project", Language: "Go", Difficulty: "Easy", EstimatedDurationInMinutes: 30},
	}
	component.SetProjects(projects)

	// Mark the project as downloaded in config
	err := configManager.UpdateDownloadedProject("test-project")
	if err != nil {
		t.Fatalf("Failed to mark project as downloaded: %v", err)
	}

	// Act
	updatedComponent, cmd := component.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Assert
	if updatedComponent == nil {
		t.Error("Expected component to be returned")
	}

	// Command might be nil if no project is highlighted (depends on table state)
	if cmd != nil {
		// Execute the command to get the message
		msg := cmd()
		// Since the directory won't exist, it should return an error message or redownload message
		switch msg.(type) {
		case ProjectRedownloadNeededMsg, ProjectsErrorMsg:
			// Both are valid outcomes when project directory is not found
		default:
			t.Errorf("Expected ProjectRedownloadNeededMsg or ProjectsErrorMsg, got %T", msg)
		}
	}
}

func TestComponent_View_Loading(t *testing.T) {
	// Arrange
	mockClient := &MockClient{}
	configManager := newTestConfigManager()
	fileManager := filesystem.NewManager()
	component := New(mockClient, configManager, fileManager)
	component.SetLoading(true)

	// Act
	view := component.View()

	// Assert
	if !strings.Contains(view, "Loading projects") {
		t.Error("Expected view to contain 'Loading projects'")
	}
}

func TestComponent_View_WithProjects(t *testing.T) {
	// Arrange
	mockClient := &MockClient{}
	configManager := newTestConfigManager()
	fileManager := filesystem.NewManager()
	component := New(mockClient, configManager, fileManager)

	projects := []api.Project{
		{ID: "1", Name: "Test Project", Language: "Go", Difficulty: "Easy", EstimatedDurationInMinutes: 30},
	}
	component.SetProjects(projects)

	// Act
	view := component.View()

	// Assert
	if view == "" {
		t.Error("Expected view to render table content")
	}
	// Note: Help text is now handled by the main TUI, not the component
}

func TestComponent_View_WithError(t *testing.T) {
	// Arrange
	mockClient := &MockClient{}
	configManager := newTestConfigManager()
	fileManager := filesystem.NewManager()
	component := New(mockClient, configManager, fileManager)
	component.SetError("test error")

	// Act
	view := component.View()

	// Assert
	if !strings.Contains(view, "test error") {
		t.Error("Expected view to contain error message")
	}
}

func TestComponent_UpdateProjectStatus(t *testing.T) {
	// Arrange
	mockClient := &MockClient{}
	configManager := newTestConfigManager()
	fileManager := filesystem.NewManager()
	component := New(mockClient, configManager, fileManager)

	// Act - This should not panic
	component.UpdateProjectStatus()

	// Assert - Just verify the method can be called without error
	// The actual update is handled by the table component
}

func TestComponent_GetSelectedProject_NoProjects(t *testing.T) {
	// Arrange
	mockClient := &MockClient{}
	configManager := newTestConfigManager()
	fileManager := filesystem.NewManager()
	component := New(mockClient, configManager, fileManager)

	// Act
	selectedProject := component.GetSelectedProject()

	// Assert
	if selectedProject != nil {
		t.Error("Expected no selected project when there are no projects")
	}
}

func TestComponent_GetSelectedProject_WithProjects(t *testing.T) {
	// Arrange
	mockClient := &MockClient{}
	configManager := newTestConfigManager()
	fileManager := filesystem.NewManager()
	component := New(mockClient, configManager, fileManager)

	projects := []api.Project{
		{ID: "1", Name: "Test Project", Language: "Go", Difficulty: "Easy", EstimatedDurationInMinutes: 30},
	}
	component.SetProjects(projects)

	// Act
	selectedProject := component.GetSelectedProject()

	// Assert
	// Note: The selected project depends on the table component's internal state
	// We mainly verify the method doesn't crash and returns the expected type
	if selectedProject != nil && selectedProject.ID != "1" {
		t.Errorf("Unexpected selected project ID, got '%s'", selectedProject.ID)
	}
}

func TestComponent_HandleDownloadedProject_DirectoryExists(t *testing.T) {
	// Arrange
	mockClient := &MockClient{}
	configManager := newTestConfigManager()
	fileManager := filesystem.NewManager()
	component := New(mockClient, configManager, fileManager)

	project := &api.Project{
		ID:                         "test-project",
		Name:                       "Test Project",
		Language:                   "Go",
		Difficulty:                 "Easy",
		EstimatedDurationInMinutes: 30,
	}

	// Act
	cmd := component.handleDownloadedProject(project)

	// Assert
	if cmd == nil {
		t.Error("Expected command to be returned")
	}

	// Execute the command - it will likely return an error since the directory doesn't exist
	// but we're testing that the method doesn't panic
	msg := cmd()

	// Check that we get some kind of message back
	if msg == nil {
		t.Error("Expected message to be returned from command")
	}
}
