package tracing

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TUIIntegration provides helpers for integrating tracing with Bubble Tea applications.
// This implements the Adapter pattern to bridge the tracing system with TUI events.
type TUIIntegration struct {
	manager *Manager
}

// NewTUIIntegration creates a new TUI integration helper
func NewTUIIntegration(manager *Manager) *TUIIntegration {
	return &TUIIntegration{
		manager: manager,
	}
}

// TrackKeyMsg tracks a Bubble Tea key message
func (t *TUIIntegration) TrackKeyMsg(msg tea.KeyMsg, currentState string) error {
	if t.manager == nil {
		return nil
	}

	keyStr := msg.String()
	return t.manager.TrackKeyPress(keyStr, currentState)
}

// TrackStateChange tracks a state transition in the TUI
func (t *TUIIntegration) TrackStateChange(oldState, newState, trigger string) error {
	if t.manager == nil {
		return nil
	}

	return t.manager.TrackStateTransition(oldState, newState, trigger)
}

// TrackMenuNavigation tracks navigation within menus
func (t *TUIIntegration) TrackMenuNavigation(menuType, action, selection string) error {
	if t.manager == nil {
		return nil
	}

	return t.manager.TrackMenuSelection(fmt.Sprintf("%s_%s", menuType, action), selection)
}

// TrackAPICall tracks API calls with timing
func (t *TUIIntegration) TrackAPICall(endpoint string) *TimedOperationTracker {
	if t.manager == nil {
		return &TimedOperationTracker{
			operation: endpoint,
			startTime: time.Now(),
		}
	}

	return t.manager.TimedOperation(fmt.Sprintf("api_call_%s", endpoint))
}

// TrackFileOperation tracks file operations
func (t *TUIIntegration) TrackFileOperation(operation, filePath string) *TimedOperationTracker {
	if t.manager == nil {
		return &TimedOperationTracker{
			operation: operation,
			startTime: time.Now(),
		}
	}

	tracker := t.manager.TimedOperation(fmt.Sprintf("file_%s", operation))
	tracker.AddMetadata("file_path", filePath)
	return tracker
}

// TrackProjectOperation tracks project-related operations
func (t *TUIIntegration) TrackProjectOperation(operation, projectName string) *TimedOperationTracker {
	if t.manager == nil {
		return &TimedOperationTracker{
			operation: operation,
			startTime: time.Now(),
		}
	}

	tracker := t.manager.TimedOperation(fmt.Sprintf("project_%s", operation))
	tracker.AddMetadata("project_name", projectName)
	return tracker
}

// TrackError tracks errors with component context
func (t *TUIIntegration) TrackError(err error, component, operation string) error {
	if t.manager == nil {
		return nil
	}

	context := map[string]string{
		"operation": operation,
	}

	return t.manager.TrackErrorWithContext(err, component, context)
}

// Example integration patterns for the 404skill CLI

// ExampleControllerIntegration shows how to integrate tracing into the controller
func ExampleControllerIntegration() {
	// Initialize tracing
	config := DefaultConfig()
	config.LocalDir = "~/.404skill/traces"

	manager, err := NewManager(config)
	if err != nil {
		fmt.Printf("Failed to initialize tracing: %v\n", err)
		return
	}
	defer manager.Close()

	integration := NewTUIIntegration(manager)

	// Example: Track state transitions
	_ = integration.TrackStateChange("main_menu", "project_list", "user_selection")

	// Example: Track API calls
	apiTracker := integration.TrackAPICall("get_projects")
	apiTracker.AddMetadata("endpoint", "/api/projects")
	// ... perform API call ...
	_ = apiTracker.Complete()

	// Example: Track project operations
	projectTracker := integration.TrackProjectOperation("download", "react-starter")
	projectTracker.AddMetadata("variant", "typescript")
	// ... perform download ...
	_ = projectTracker.Complete()
}

// ExampleKeyHandling shows how to track key presses in update functions
func ExampleKeyHandling() {
	config := DefaultConfig()
	manager, _ := NewManager(config)
	defer manager.Close()

	integration := NewTUIIntegration(manager)

	// In your Bubble Tea Update function:
	updateFunc := func(msg tea.Msg, currentState string) tea.Cmd {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			// Track the key press
			_ = integration.TrackKeyMsg(msg, currentState)

			switch msg.String() {
			case "q", "ctrl+c":
				_ = integration.TrackStateChange(currentState, "exit", "quit_key")
				return tea.Quit
			case "enter":
				_ = integration.TrackStateChange(currentState, "next_state", "enter_key")
				// Handle enter...
			}
		}
		return nil
	}

	// Use the update function...
	_ = updateFunc
}

// ExampleErrorTracking shows how to track errors throughout the application
func ExampleErrorTracking() {
	config := DefaultConfig()
	manager, _ := NewManager(config)
	defer manager.Close()

	integration := NewTUIIntegration(manager)

	// Example: Track API errors
	err := fmt.Errorf("failed to fetch projects: connection timeout")
	_ = integration.TrackError(err, "api_client", "fetch_projects")

	// Example: Track file system errors
	err = fmt.Errorf("permission denied")
	context := map[string]string{
		"operation": "create_directory",
		"path":      "/restricted/path",
	}
	_ = manager.TrackErrorWithContext(err, "file_system", context)
}

// ExamplePerformanceTracking shows comprehensive performance tracking
func ExamplePerformanceTracking() {
	config := DefaultConfig()
	manager, _ := NewManager(config)
	defer manager.Close()

	// Track overall application performance
	appTracker := manager.TimedOperation("application_startup")
	appTracker.AddMetadata("version", "1.0.0")

	// Track component initialization
	initTracker := manager.TimedOperation("component_init")
	initTracker.AddMetadata("component", "project_service")
	// ... initialize component ...
	_ = initTracker.Complete()

	// Track user workflows
	workflowTracker := manager.TimedOperation("user_workflow_download_project")
	workflowTracker.AddMetadata("project_type", "react")
	workflowTracker.AddMetadata("variant", "typescript")
	// ... complete workflow ...
	_ = workflowTracker.Complete()

	// Complete application startup
	_ = appTracker.Complete()
}

// IntegrationMiddleware provides middleware for automatic tracing
type IntegrationMiddleware struct {
	integration  *TUIIntegration
	currentState string
}

// NewIntegrationMiddleware creates tracing middleware
func NewIntegrationMiddleware(integration *TUIIntegration) *IntegrationMiddleware {
	return &IntegrationMiddleware{
		integration: integration,
	}
}

// WrapUpdate wraps a Bubble Tea update function with automatic tracing
func (m *IntegrationMiddleware) WrapUpdate(updateFunc func(tea.Msg) (tea.Model, tea.Cmd)) func(tea.Msg) (tea.Model, tea.Cmd) {
	return func(msg tea.Msg) (tea.Model, tea.Cmd) {
		// Track the message
		switch msg := msg.(type) {
		case tea.KeyMsg:
			_ = m.integration.TrackKeyMsg(msg, m.currentState)
		}

		// Call the original update function
		model, cmd := updateFunc(msg)

		return model, cmd
	}
}

// SetCurrentState updates the current state for tracking
func (m *IntegrationMiddleware) SetCurrentState(state string) {
	oldState := m.currentState
	m.currentState = state

	if oldState != "" && oldState != state {
		_ = m.integration.TrackStateChange(oldState, state, "state_update")
	}
}
