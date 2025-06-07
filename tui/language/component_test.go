package language

import (
	"404skill-cli/api"
	"404skill-cli/downloader"
	"404skill-cli/tui/components/menu"
	"context"
	"errors"
	"strings"
	"testing"
)

// MockDownloader implements downloader.Downloader for testing
type MockDownloader struct {
	downloadProjectFunc func(ctx context.Context, project *api.Project, language string, progressCallback downloader.ProgressCallback) error
}

func (m *MockDownloader) DownloadProject(ctx context.Context, project *api.Project, language string, progressCallback downloader.ProgressCallback) error {
	if m.downloadProjectFunc != nil {
		return m.downloadProjectFunc(ctx, project, language, progressCallback)
	}
	return nil
}

func TestComponent_New(t *testing.T) {
	// Arrange
	project := &api.Project{
		ID:       "test-project",
		Name:     "Test Project",
		Language: "Go, Python, JavaScript",
	}
	mockDownloader := &MockDownloader{}

	// Act
	component := New(project, mockDownloader)

	// Assert
	if component == nil {
		t.Error("Expected component to be created")
	}
	if component.project != project {
		t.Error("Expected project to be set")
	}
	if component.downloader != mockDownloader {
		t.Error("Expected downloader to be set")
	}
	if component.menu == nil {
		t.Error("Expected menu to be created")
	}
	if component.downloading {
		t.Error("Expected downloading to be false initially")
	}
}

func TestComponent_SetProject(t *testing.T) {
	// Arrange
	initialProject := &api.Project{
		ID:       "initial-project",
		Name:     "Initial Project",
		Language: "Go",
	}
	mockDownloader := &MockDownloader{}
	component := New(initialProject, mockDownloader)

	newProject := &api.Project{
		ID:       "new-project",
		Name:     "New Project",
		Language: "Python, JavaScript, TypeScript",
	}

	// Act
	component.SetProject(newProject)

	// Assert
	if component.project != newProject {
		t.Error("Expected project to be updated")
	}
	if component.downloading {
		t.Error("Expected downloading to be reset to false")
	}
	if component.progress != 0 {
		t.Error("Expected progress to be reset to 0")
	}
	if component.errorMsg != "" {
		t.Error("Expected error message to be cleared")
	}

	// Check that menu was updated with new languages
	selectedLanguage := component.GetSelectedLanguage()
	if selectedLanguage != "Python" {
		t.Errorf("Expected first language to be 'Python', got '%s'", selectedLanguage)
	}
}

func TestComponent_SetDownloading(t *testing.T) {
	// Arrange
	project := &api.Project{ID: "test", Name: "Test", Language: "Go"}
	mockDownloader := &MockDownloader{}
	component := New(project, mockDownloader)

	// Act - Set downloading to true
	component.SetDownloading(true)

	// Assert
	if !component.downloading {
		t.Error("Expected downloading to be true")
	}

	// Act - Set downloading to false
	component.SetDownloading(false)

	// Assert
	if component.downloading {
		t.Error("Expected downloading to be false")
	}
	if component.progress != 0 {
		t.Error("Expected progress to be reset when downloading stops")
	}
}

func TestComponent_SetProgress(t *testing.T) {
	// Arrange
	project := &api.Project{ID: "test", Name: "Test", Language: "Go"}
	mockDownloader := &MockDownloader{}
	component := New(project, mockDownloader)

	// Act
	component.SetProgress(0.75)

	// Assert
	if component.progress != 0.75 {
		t.Errorf("Expected progress to be 0.75, got %f", component.progress)
	}
}

func TestComponent_SetError(t *testing.T) {
	// Arrange
	project := &api.Project{ID: "test", Name: "Test", Language: "Go"}
	mockDownloader := &MockDownloader{}
	component := New(project, mockDownloader)
	component.SetDownloading(true)

	// Act
	component.SetError("test error")

	// Assert
	if component.errorMsg != "test error" {
		t.Errorf("Expected error 'test error', got '%s'", component.errorMsg)
	}
	if component.downloading {
		t.Error("Expected downloading to be false after error")
	}
}

func TestComponent_GetSelectedLanguage(t *testing.T) {
	// Arrange
	project := &api.Project{
		ID:       "test-project",
		Name:     "Test Project",
		Language: "Go, Python, JavaScript",
	}
	mockDownloader := &MockDownloader{}
	component := New(project, mockDownloader)

	// Act
	selectedLanguage := component.GetSelectedLanguage()

	// Assert
	if selectedLanguage != "Go" {
		t.Errorf("Expected selected language to be 'Go', got '%s'", selectedLanguage)
	}
}

func TestComponent_Update_MenuSelect_SuccessfulDownload(t *testing.T) {
	// Arrange
	project := &api.Project{
		ID:       "test-project",
		Name:     "Test Project",
		Language: "Go, Python",
	}
	mockDownloader := &MockDownloader{
		downloadProjectFunc: func(ctx context.Context, project *api.Project, language string, progressCallback downloader.ProgressCallback) error {
			return nil // Successful download
		},
	}
	component := New(project, mockDownloader)

	// Act - Simulate menu selection by sending MenuSelectMsg directly
	updatedComponent, cmd := component.Update(menu.MenuSelectMsg{
		SelectedIndex: 0,
		SelectedItem:  "Go",
	})

	// Assert
	if updatedComponent == nil {
		t.Error("Expected component to be returned")
	}
	if !updatedComponent.downloading {
		t.Error("Expected downloading to be true after selection")
	}
	if cmd == nil {
		t.Error("Expected command to be returned")
	}

	// Execute the command to simulate download
	msg := cmd()
	if completeMsg, ok := msg.(DownloadCompleteMsg); ok {
		if completeMsg.Project != project {
			t.Error("Expected project in completion message")
		}
		if completeMsg.Language != "Go" {
			t.Errorf("Expected language 'Go', got '%s'", completeMsg.Language)
		}
	} else {
		t.Errorf("Expected DownloadCompleteMsg, got %T", msg)
	}
}

func TestComponent_Update_MenuSelect_DownloadError(t *testing.T) {
	// Arrange
	project := &api.Project{
		ID:       "test-project",
		Name:     "Test Project",
		Language: "Go, Python",
	}
	mockDownloader := &MockDownloader{
		downloadProjectFunc: func(ctx context.Context, project *api.Project, language string, progressCallback downloader.ProgressCallback) error {
			return errors.New("download failed")
		},
	}
	component := New(project, mockDownloader)

	// Act - Simulate menu selection by sending MenuSelectMsg directly
	updatedComponent, cmd := component.Update(menu.MenuSelectMsg{
		SelectedIndex: 0,
		SelectedItem:  "Go",
	})

	// Assert
	if updatedComponent == nil {
		t.Error("Expected component to be returned")
	}
	if cmd == nil {
		t.Error("Expected command to be returned")
	}

	// Execute the command to simulate download
	msg := cmd()
	if errorMsg, ok := msg.(DownloadErrorMsg); ok {
		if errorMsg.Error != "download failed" {
			t.Errorf("Expected error 'download failed', got '%s'", errorMsg.Error)
		}
	} else {
		t.Errorf("Expected DownloadErrorMsg, got %T", msg)
	}
}

func TestComponent_Update_DownloadProgressMsg(t *testing.T) {
	// Arrange
	project := &api.Project{ID: "test", Name: "Test", Language: "Go"}
	mockDownloader := &MockDownloader{}
	component := New(project, mockDownloader)

	// Act
	updatedComponent, cmd := component.Update(DownloadProgressMsg{Progress: 0.6})

	// Assert
	if updatedComponent == nil {
		t.Error("Expected component to be returned")
	}
	if updatedComponent.progress != 0.6 {
		t.Errorf("Expected progress to be 0.6, got %f", updatedComponent.progress)
	}
	if cmd != nil {
		t.Error("Expected no command for progress update")
	}
}

func TestComponent_Update_DownloadCompleteMsg(t *testing.T) {
	// Arrange
	project := &api.Project{ID: "test", Name: "Test", Language: "Go"}
	mockDownloader := &MockDownloader{}
	component := New(project, mockDownloader)
	component.SetDownloading(true)

	// Act
	updatedComponent, cmd := component.Update(DownloadCompleteMsg{
		Project:  project,
		Language: "Go",
	})

	// Assert
	if updatedComponent == nil {
		t.Error("Expected component to be returned")
	}
	if updatedComponent.downloading {
		t.Error("Expected downloading to be false after completion")
	}
	if cmd != nil {
		t.Error("Expected no command for completion message")
	}
}

func TestComponent_Update_DownloadErrorMsg(t *testing.T) {
	// Arrange
	project := &api.Project{ID: "test", Name: "Test", Language: "Go"}
	mockDownloader := &MockDownloader{}
	component := New(project, mockDownloader)
	component.SetDownloading(true)

	// Act
	updatedComponent, cmd := component.Update(DownloadErrorMsg{Error: "download failed"})

	// Assert
	if updatedComponent == nil {
		t.Error("Expected component to be returned")
	}
	if updatedComponent.downloading {
		t.Error("Expected downloading to be false after error")
	}
	if updatedComponent.errorMsg != "download failed" {
		t.Errorf("Expected error 'download failed', got '%s'", updatedComponent.errorMsg)
	}
	if cmd != nil {
		t.Error("Expected no command for error message")
	}
}

func TestComponent_View_Normal(t *testing.T) {
	// Arrange
	project := &api.Project{
		ID:       "test-project",
		Name:     "Test Project",
		Language: "Go, Python",
	}
	mockDownloader := &MockDownloader{}
	component := New(project, mockDownloader)

	// Act
	view := component.View()

	// Assert
	if !strings.Contains(view, "Test Project") {
		t.Error("Expected view to contain project name")
	}
	if !strings.Contains(view, "Use ↑/↓ or k/j to move") {
		t.Error("Expected view to contain help text")
	}
}

func TestComponent_View_Downloading(t *testing.T) {
	// Arrange
	project := &api.Project{ID: "test", Name: "Test", Language: "Go"}
	mockDownloader := &MockDownloader{}
	component := New(project, mockDownloader)
	component.SetDownloading(true)
	component.SetProgress(0.5)

	// Act
	view := component.View()

	// Assert
	if !strings.Contains(view, "Downloading project") {
		t.Error("Expected view to contain downloading text")
	}
	if !strings.Contains(view, "50%") {
		t.Error("Expected view to contain progress percentage")
	}
}

func TestComponent_View_WithError(t *testing.T) {
	// Arrange
	project := &api.Project{ID: "test", Name: "Test", Language: "Go"}
	mockDownloader := &MockDownloader{}
	component := New(project, mockDownloader)
	component.SetError("test error")

	// Act
	view := component.View()

	// Assert
	if !strings.Contains(view, "test error") {
		t.Error("Expected view to contain error message")
	}
}
