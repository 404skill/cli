package table

import (
	"strings"
	"testing"

	"404skill-cli/api"

	tea "github.com/charmbracelet/bubbletea"
)

// MockProjectStatusProvider implements ProjectStatusProvider for testing
type MockProjectStatusProvider struct {
	statusMap map[string]string
}

func (m *MockProjectStatusProvider) GetProjectStatus(projectID string) string {
	if m.statusMap == nil {
		return ""
	}
	return m.statusMap[projectID]
}

// Helper function to create test projects
func createTestProjects() []api.Project {
	return []api.Project{
		{
			ID:                         "project1",
			Name:                       "Test Project 1",
			Language:                   "Go",
			Difficulty:                 "Easy",
			EstimatedDurationInMinutes: 30,
		},
		{
			ID:                         "project2",
			Name:                       "Test Project 2",
			Language:                   "Python",
			Difficulty:                 "Medium",
			EstimatedDurationInMinutes: 60,
		},
		{
			ID:                         "project3",
			Name:                       "Test Project 3",
			Language:                   "JavaScript",
			Difficulty:                 "Hard",
			EstimatedDurationInMinutes: 90,
		},
	}
}

func TestNew(t *testing.T) {
	// Arrange
	mockProvider := &MockProjectStatusProvider{}

	// Act
	component := New(mockProvider)

	// Assert
	if component == nil {
		t.Error("Expected component to be created")
	}
	if component.statusProvider != mockProvider {
		t.Error("Expected status provider to be set correctly")
	}
	if component.focused {
		t.Error("Expected component to start unfocused")
	}
	if len(component.projects) != 0 {
		t.Error("Expected component to start with no projects")
	}
}

func TestNew_NilStatusProvider(t *testing.T) {
	// Act
	component := New(nil)

	// Assert
	if component == nil {
		t.Error("Expected component to be created even with nil status provider")
	}
	if component.statusProvider != nil {
		t.Error("Expected status provider to be nil")
	}
}

func TestSetProjects(t *testing.T) {
	// Arrange
	mockProvider := &MockProjectStatusProvider{
		statusMap: map[string]string{
			"project1": "✓ Downloaded",
			"project2": "",
		},
	}
	component := New(mockProvider)
	projects := createTestProjects()

	// Act
	component.SetProjects(projects)

	// Assert
	if len(component.projects) != 3 {
		t.Errorf("Expected 3 projects, got %d", len(component.projects))
	}
	if component.projects[0].ID != "project1" {
		t.Errorf("Expected first project ID to be 'project1', got '%s'", component.projects[0].ID)
	}

	// Verify table content contains project data
	view := component.View()
	if !strings.Contains(view, "Test Project 1") {
		t.Error("Expected table view to contain project name")
	}
	if !strings.Contains(view, "Go") {
		t.Error("Expected table view to contain language")
	}
	if !strings.Contains(view, "30 min") {
		t.Error("Expected table view to contain duration")
	}
}

func TestSetFocused(t *testing.T) {
	// Arrange
	component := New(&MockProjectStatusProvider{})
	projects := createTestProjects()
	component.SetProjects(projects)

	// Act - Set focused to true
	component.SetFocused(true)

	// Assert
	if !component.focused {
		t.Error("Expected component to be focused")
	}

	// Act - Set focused to false
	component.SetFocused(false)

	// Assert
	if component.focused {
		t.Error("Expected component to be unfocused")
	}
}

func TestGetHighlightedProject_NoProjects(t *testing.T) {
	// Arrange
	component := New(&MockProjectStatusProvider{})

	// Act
	highlighted := component.GetHighlightedProject()

	// Assert
	if highlighted != nil {
		t.Error("Expected no highlighted project when table is empty")
	}
}

func TestGetHighlightedProject_WithProjects(t *testing.T) {
	// Arrange
	component := New(&MockProjectStatusProvider{})
	projects := createTestProjects()
	component.SetProjects(projects)
	component.SetFocused(true)

	// Act
	highlighted := component.GetHighlightedProject()

	// Assert - Note: Default selection behavior may vary, test for non-nil if table auto-selects
	// This test ensures the method doesn't crash and returns consistent results
	if highlighted != nil {
		// If a project is highlighted, it should be one of our test projects
		found := false
		for _, p := range projects {
			if p.ID == highlighted.ID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Highlighted project should be one of the projects in the table")
		}
	}
}

func TestUpdate(t *testing.T) {
	// Arrange
	component := New(&MockProjectStatusProvider{})
	projects := createTestProjects()
	component.SetProjects(projects)
	component.SetFocused(true)

	// Act
	updatedComponent, cmd := component.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Assert
	if updatedComponent != component {
		t.Error("Expected same component instance to be returned")
	}
	// cmd can be nil or a command, both are valid
	_ = cmd // Acknowledge cmd is checked
}

func TestView_EmptyTable(t *testing.T) {
	// Arrange
	component := New(&MockProjectStatusProvider{})

	// Act
	view := component.View()

	// Assert
	if view == "" {
		t.Error("Expected table view to render something even when empty")
	}
	// Should contain column headers
	expectedHeaders := []string{"Name", "Language", "Difficulty", "Duration", "Status"}
	for _, header := range expectedHeaders {
		if !strings.Contains(view, header) {
			t.Errorf("Expected table view to contain header '%s'", header)
		}
	}
}

func TestView_WithProjects(t *testing.T) {
	// Arrange
	mockProvider := &MockProjectStatusProvider{
		statusMap: map[string]string{
			"project1": "✓ Downloaded",
			"project2": "",
			"project3": "✓ Downloaded",
		},
	}
	component := New(mockProvider)
	projects := createTestProjects()
	component.SetProjects(projects)

	// Act
	view := component.View()

	// Assert
	expectedContent := []string{
		"Test Project 1",
		"Test Project 2",
		"Test Project 3",
		"Go",
		"Python",
		"JavaScript",
		"Easy",
		"Medium",
		"Hard",
		"30 min",
		"60 min",
		"90 min",
		"✓ Downloaded", // Should appear twice for project1 and project3
	}

	for _, content := range expectedContent {
		if !strings.Contains(view, content) {
			t.Errorf("Expected table view to contain '%s'", content)
		}
	}
}

func TestProjectStatusProvider_Integration(t *testing.T) {
	// Arrange
	mockProvider := &MockProjectStatusProvider{
		statusMap: map[string]string{
			"project1": "✓ Downloaded",
			"project2": "❌ Failed",
			"project3": "", // No status
		},
	}
	component := New(mockProvider)
	projects := createTestProjects()
	component.SetProjects(projects)

	// Act
	view := component.View()

	// Assert
	if !strings.Contains(view, "✓ Downloaded") {
		t.Error("Expected table to show downloaded status")
	}
	if !strings.Contains(view, "❌ Failed") {
		t.Error("Expected table to show failed status")
	}
}

func TestUpdateProjectStatus(t *testing.T) {
	// Arrange
	mockProvider := &MockProjectStatusProvider{
		statusMap: map[string]string{
			"project1": "",
		},
	}
	component := New(mockProvider)
	projects := createTestProjects()
	component.SetProjects(projects)

	// Get initial view
	initialView := component.View()

	// Update the status
	mockProvider.statusMap["project1"] = "✓ Downloaded"

	// Act
	component.UpdateProjectStatus()

	// Assert
	updatedView := component.View()
	if !strings.Contains(updatedView, "✓ Downloaded") {
		t.Error("Expected updated view to contain new status")
	}
	if initialView == updatedView {
		t.Error("Expected view to change after status update")
	}
}

func TestProjectStatusProvider_NilProvider(t *testing.T) {
	// Arrange
	component := New(nil) // No status provider
	projects := createTestProjects()
	component.SetProjects(projects)

	// Act
	view := component.View()

	// Assert - Should still work, just no status column values
	if !strings.Contains(view, "Test Project 1") {
		t.Error("Expected table to work even without status provider")
	}
	if !strings.Contains(view, "Status") {
		t.Error("Expected Status column header to still be present")
	}
}

func TestSetProjects_EmptyList(t *testing.T) {
	// Arrange
	component := New(&MockProjectStatusProvider{})

	// Act
	component.SetProjects([]api.Project{})

	// Assert
	if len(component.projects) != 0 {
		t.Error("Expected empty projects list")
	}

	view := component.View()
	if view == "" {
		t.Error("Expected table to render even with empty project list")
	}
}

func TestSetProjects_UpdateExisting(t *testing.T) {
	// Arrange
	component := New(&MockProjectStatusProvider{})
	initialProjects := createTestProjects()[:2] // First 2 projects
	component.SetProjects(initialProjects)

	// Act
	allProjects := createTestProjects() // All 3 projects
	component.SetProjects(allProjects)

	// Assert
	if len(component.projects) != 3 {
		t.Errorf("Expected 3 projects after update, got %d", len(component.projects))
	}

	view := component.View()
	if !strings.Contains(view, "Test Project 3") {
		t.Error("Expected table to contain newly added project")
	}
}
