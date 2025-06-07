package test

import (
	"404skill-cli/api"
	"404skill-cli/testreport"
	"404skill-cli/testrunner"
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

// TestCompleteMsg is sent when testing is complete
type TestCompleteMsg struct {
	Project *testrunner.Project
	Result  *testreport.ParseResult
	Error   string
}

// TestProgressMsg is sent during test execution
type TestProgressMsg struct {
	Line string
}

// TestErrorMsg is sent when testing fails
type TestErrorMsg struct {
	Error string
}

// ConfigManager interface for project configuration
type ConfigManager interface {
	IsProjectDownloaded(projectID string) bool
}

// APIClient interface for updating test results
type APIClient interface {
	BulkUpdateProfileTests(ctx context.Context, failed []string, passed []string, projectID string) error
}

// Component interface for tea components
type Component interface {
	Init() tea.Cmd
	Update(tea.Msg) (Component, tea.Cmd)
	View() string
	SetProjects([]api.Project)
}
