package test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"404skill-cli/api"
	"404skill-cli/testreport"
	"404skill-cli/testrunner"

	tea "github.com/charmbracelet/bubbletea"
)

// Mock implementations for testing
type MockTestRunner struct {
	runTestsFunc func(project testrunner.Project, progressCallback func(string)) (*testreport.ParseResult, error)
}

func (m *MockTestRunner) RunTests(project testrunner.Project, progressCallback func(string)) (*testreport.ParseResult, error) {
	if m.runTestsFunc != nil {
		return m.runTestsFunc(project, progressCallback)
	}
	return nil, nil
}

type MockConfigManager struct {
	isProjectDownloadedFunc func(projectID string) bool
}

func (m *MockConfigManager) IsProjectDownloaded(projectID string) bool {
	if m.isProjectDownloadedFunc != nil {
		return m.isProjectDownloadedFunc(projectID)
	}
	return false
}

type MockAPIClient struct {
	bulkUpdateProfileTestsFunc func(ctx context.Context, failed []string, passed []string, projectID string) error
}

func (m *MockAPIClient) BulkUpdateProfileTests(ctx context.Context, failed []string, passed []string, projectID string) error {
	if m.bulkUpdateProfileTestsFunc != nil {
		return m.bulkUpdateProfileTestsFunc(ctx, failed, passed, projectID)
	}
	return nil
}

func TestTestComponent_New(t *testing.T) {
	testRunner := &MockTestRunner{}
	configManager := &MockConfigManager{}
	apiClient := &MockAPIClient{}

	component := New(testRunner, configManager, apiClient)

	if component == nil {
		t.Fatal("Expected component to be created")
	}

	if component.testRunner != testRunner {
		t.Error("Expected testRunner to be set")
	}

	if component.configManager != configManager {
		t.Error("Expected configManager to be set")
	}

	if component.apiClient != apiClient {
		t.Error("Expected apiClient to be set")
	}

	if component.testing {
		t.Error("Expected testing to be false initially")
	}

	if component.showingTestResults {
		t.Error("Expected showingTestResults to be false initially")
	}
}

func TestTestComponent_SetProjects(t *testing.T) {
	configManager := &MockConfigManager{
		isProjectDownloadedFunc: func(projectID string) bool {
			return projectID == "downloaded-project"
		},
	}

	component := New(&MockTestRunner{}, configManager, &MockAPIClient{})

	projects := []api.Project{
		{
			ID:                         "downloaded-project",
			Name:                       "Downloaded Project",
			Language:                   "go",
			Difficulty:                 "Medium",
			EstimatedDurationInMinutes: 30,
		},
		{
			ID:                         "not-downloaded-project",
			Name:                       "Not Downloaded Project",
			Language:                   "python",
			Difficulty:                 "Easy",
			EstimatedDurationInMinutes: 15,
		},
	}

	component.SetProjects(projects)

	if len(component.projects) != 1 {
		t.Errorf("Expected 1 project, got %d", len(component.projects))
	}

	if component.projects[0].ID != "downloaded-project" {
		t.Error("Expected downloaded project to be included")
	}
}

func TestTestComponent_Update_KeyHandling(t *testing.T) {
	tests := []struct {
		name           string
		initialState   func(*TestComponent)
		keyMsg         string
		expectedAction string
	}{
		{
			name: "enter key starts test",
			initialState: func(c *TestComponent) {
				c.SetProjects([]api.Project{
					{
						ID:       "test-project",
						Name:     "Test Project",
						Language: "go",
					},
				})
			},
			keyMsg:         "enter",
			expectedAction: "start_test",
		},
		{
			name: "any key dismisses test results",
			initialState: func(c *TestComponent) {
				c.showingTestResults = true
				c.testResultsSummary = "Test Results"
			},
			keyMsg:         "space",
			expectedAction: "dismiss_results",
		},
		{
			name: "keys ignored during testing",
			initialState: func(c *TestComponent) {
				c.testing = true
			},
			keyMsg:         "enter",
			expectedAction: "ignore",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configManager := &MockConfigManager{
				isProjectDownloadedFunc: func(projectID string) bool {
					return true
				},
			}

			testRunner := &MockTestRunner{
				runTestsFunc: func(project testrunner.Project, progressCallback func(string)) (*testreport.ParseResult, error) {
					return &testreport.ParseResult{
						Suite: testreport.TestSuite{Name: "Test Suite"},
					}, nil
				},
			}

			component := New(testRunner, configManager, &MockAPIClient{})
			tt.initialState(component)

			keyMsg := tea.KeyMsg{
				Type:  tea.KeyRunes,
				Runes: []rune(tt.keyMsg),
			}

			updatedComponent, cmd := component.Update(keyMsg)
			component = updatedComponent.(*TestComponent)

			switch tt.expectedAction {
			case "start_test":
				if !component.testing {
					t.Error("Expected testing to be true after enter key")
				}
				if cmd == nil {
					t.Error("Expected command to be returned for starting test")
				}
			case "dismiss_results":
				if component.showingTestResults {
					t.Error("Expected showingTestResults to be false after key press")
				}
				if component.testResultsSummary != "" {
					t.Error("Expected testResultsSummary to be cleared")
				}
			case "ignore":
				if cmd != nil {
					t.Error("Expected no command when testing is in progress")
				}
			}
		})
	}
}

func TestTestComponent_Update_TestMessages(t *testing.T) {
	tests := []struct {
		name            string
		message         tea.Msg
		expectedTesting bool
		expectedError   string
		expectedResults bool
	}{
		{
			name: "test complete with success",
			message: TestCompleteMsg{
				Project: &testrunner.Project{ID: "test-project"},
				Result: &testreport.ParseResult{
					Suite: testreport.TestSuite{
						Name:  "Test Suite",
						Tests: 3,
						Time:  1.5,
					},
					PassedTests: []string{"test1", "test2"},
					FailedTests: []string{"test3"},
				},
			},
			expectedTesting: false,
			expectedResults: true,
		},
		{
			name: "test complete with error",
			message: TestCompleteMsg{
				Project: &testrunner.Project{ID: "test-project"},
				Error:   "Test execution failed",
			},
			expectedTesting: false,
			expectedError:   "Test execution failed",
			expectedResults: false,
		},
		{
			name: "test progress message",
			message: TestProgressMsg{
				Line: "Running test 1...",
			},
			expectedTesting: true, // Should not change testing state
		},
		{
			name: "test error message",
			message: TestErrorMsg{
				Error: "Docker compose failed",
			},
			expectedTesting: false,
			expectedError:   "Docker compose failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiClient := &MockAPIClient{
				bulkUpdateProfileTestsFunc: func(ctx context.Context, failed []string, passed []string, projectID string) error {
					return nil
				},
			}

			component := New(&MockTestRunner{}, &MockConfigManager{}, apiClient)
			component.testing = true // Set initial testing state

			updatedComponent, _ := component.Update(tt.message)
			component = updatedComponent.(*TestComponent)

			if component.testing != tt.expectedTesting {
				t.Errorf("Expected testing=%v, got %v", tt.expectedTesting, component.testing)
			}

			if component.errorMsg != tt.expectedError {
				t.Errorf("Expected error=%q, got %q", tt.expectedError, component.errorMsg)
			}

			if component.showingTestResults != tt.expectedResults {
				t.Errorf("Expected showingTestResults=%v, got %v", tt.expectedResults, component.showingTestResults)
			}
		})
	}
}

func TestTestComponent_Update_SpinnerMessages(t *testing.T) {
	component := New(&MockTestRunner{}, &MockConfigManager{}, &MockAPIClient{})
	component.testing = true

	spinnerMsg := spinnerMsg{frame: "⠙"}
	updatedComponent, cmd := component.Update(spinnerMsg)
	component = updatedComponent.(*TestComponent)

	if component.spinnerFrame != "⠙" {
		t.Errorf("Expected spinner frame to be updated to ⠙, got %s", component.spinnerFrame)
	}

	if cmd == nil {
		t.Error("Expected spinner tick command when testing is active")
	}
}

func TestTestComponent_Update_APIUpdateMessages(t *testing.T) {
	tests := []struct {
		name            string
		message         apiUpdateCompleteMsg
		expectedSummary string
	}{
		{
			name:            "successful API update",
			message:         apiUpdateCompleteMsg{err: nil},
			expectedSummary: "[API update successful!]",
		},
		{
			name:            "failed API update",
			message:         apiUpdateCompleteMsg{err: errors.New("connection failed")},
			expectedSummary: "[API update failed: connection failed]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component := New(&MockTestRunner{}, &MockConfigManager{}, &MockAPIClient{})
			component.testResultsSummary = "Initial summary"

			updatedComponent, _ := component.Update(tt.message)
			component = updatedComponent.(*TestComponent)

			if !strings.Contains(component.testResultsSummary, tt.expectedSummary) {
				t.Errorf("Expected summary to contain %q, got %q", tt.expectedSummary, component.testResultsSummary)
			}
		})
	}
}

func TestTestComponent_View_States(t *testing.T) {
	tests := []struct {
		name         string
		setupState   func(*TestComponent)
		expectedText []string
	}{
		{
			name: "showing test results",
			setupState: func(c *TestComponent) {
				c.showingTestResults = true
				c.testResultsSummary = "Test Results Summary"
				c.testResultsList = []string{"[PASS] test1", "[FAIL] test2"}
			},
			expectedText: []string{"Test Results Summary", "[PASS] test1", "[FAIL] test2", "Press any key"},
		},
		{
			name: "testing in progress",
			setupState: func(c *TestComponent) {
				c.testing = true
				c.spinnerFrame = "⠋"
				c.outputBuffer = []string{"Starting tests...", "Running test 1..."}
			},
			expectedText: []string{"Testing Project", "Running tests...", "⠋", "Starting tests...", "Running test 1..."},
		},
		{
			name: "showing project table",
			setupState: func(c *TestComponent) {
				c.SetProjects([]api.Project{
					{
						ID:       "test-project",
						Name:     "Test Project",
						Language: "go",
					},
				})
			},
			expectedText: []string{"select", "back", "quit"},
		},
		{
			name: "showing error message",
			setupState: func(c *TestComponent) {
				c.errorMsg = "Something went wrong"
			},
			expectedText: []string{"Something went wrong"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configManager := &MockConfigManager{
				isProjectDownloadedFunc: func(projectID string) bool {
					return true
				},
			}

			component := New(&MockTestRunner{}, configManager, &MockAPIClient{})
			tt.setupState(component)

			view := component.View()

			for _, expectedText := range tt.expectedText {
				if !strings.Contains(view, expectedText) {
					t.Errorf("Expected view to contain %q, got:\n%s", expectedText, view)
				}
			}
		})
	}
}

func TestTestComponent_buildTestResultsView(t *testing.T) {
	component := New(&MockTestRunner{}, &MockConfigManager{}, &MockAPIClient{})

	result := &testreport.ParseResult{
		Suite: testreport.TestSuite{
			Name:  "Test Suite",
			Tests: 3,
			Time:  1.5,
		},
		PassedTests: []string{"test1", "test2"},
		FailedTests: []string{"test3"},
	}
	result.Suite.Results = []testreport.TestResult{
		{Name: "test1", Passed: true, Time: 0.5},
		{Name: "test2", Passed: true, Time: 0.3},
		{Name: "test3", Passed: false, Time: 0.7},
	}

	component.buildTestResultsView(result)

	expectedSummary := "Test Results: Test Suite"
	if !strings.Contains(component.testResultsSummary, expectedSummary) {
		t.Errorf("Expected summary to contain %q, got %q", expectedSummary, component.testResultsSummary)
	}

	if !strings.Contains(component.testResultsSummary, "Total: 3") {
		t.Error("Expected summary to contain total count")
	}

	if !strings.Contains(component.testResultsSummary, "Passed: 2") {
		t.Error("Expected summary to contain passed count")
	}

	if !strings.Contains(component.testResultsSummary, "Failed: 1") {
		t.Error("Expected summary to contain failed count")
	}

	if len(component.testResultsList) != 3 {
		t.Errorf("Expected 3 test results, got %d", len(component.testResultsList))
	}

	passedCount := 0
	failedCount := 0
	for _, result := range component.testResultsList {
		if strings.Contains(result, "[PASS]") {
			passedCount++
		} else if strings.Contains(result, "[FAIL]") {
			failedCount++
		}
	}

	if passedCount != 2 {
		t.Errorf("Expected 2 passed tests in results list, got %d", passedCount)
	}

	if failedCount != 1 {
		t.Errorf("Expected 1 failed test in results list, got %d", failedCount)
	}
}

func TestTestComponent_Init(t *testing.T) {
	component := New(&MockTestRunner{}, &MockConfigManager{}, &MockAPIClient{})

	cmd := component.Init()

	if cmd != nil {
		t.Error("Expected Init to return nil command")
	}
}

func TestTestComponent_Integration(t *testing.T) {
	// Integration test that simulates a complete test cycle
	var apiCallMade bool

	testRunner := &MockTestRunner{
		runTestsFunc: func(project testrunner.Project, progressCallback func(string)) (*testreport.ParseResult, error) {
			progressCallback("Starting tests...")
			time.Sleep(10 * time.Millisecond) // Simulate test execution
			progressCallback("Tests completed")

			return &testreport.ParseResult{
				Suite: testreport.TestSuite{
					Name:  "Integration Test Suite",
					Tests: 2,
					Time:  0.5,
				},
				PassedTests: []string{"integration_test_1"},
				FailedTests: []string{"integration_test_2"},
			}, nil
		},
	}

	configManager := &MockConfigManager{
		isProjectDownloadedFunc: func(projectID string) bool {
			return projectID == "integration-project"
		},
	}

	apiClient := &MockAPIClient{
		bulkUpdateProfileTestsFunc: func(ctx context.Context, failed []string, passed []string, projectID string) error {
			apiCallMade = true
			if len(failed) != 1 || len(passed) != 1 {
				return fmt.Errorf("expected 1 failed and 1 passed test")
			}
			return nil
		},
	}

	component := New(testRunner, configManager, apiClient)

	// Set up project
	projects := []api.Project{
		{
			ID:       "integration-project",
			Name:     "Integration Project",
			Language: "go",
		},
	}
	component.SetProjects(projects)

	// Simulate user pressing enter to start test
	keyMsg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("enter"),
	}

	updatedComponent, cmd := component.Update(keyMsg)
	component = updatedComponent.(*TestComponent)

	// Verify test started
	if !component.testing {
		t.Error("Expected testing to be true after enter key")
	}

	if cmd == nil {
		t.Error("Expected command to be returned for starting test")
	}

	// The actual test execution would happen asynchronously
	// For this test, we'll simulate the completion message
	completeMsg := TestCompleteMsg{
		Project: &testrunner.Project{ID: "integration-project"},
		Result: &testreport.ParseResult{
			Suite: testreport.TestSuite{
				Name:  "Integration Test Suite",
				Tests: 2,
				Time:  0.5,
			},
			PassedTests: []string{"integration_test_1"},
			FailedTests: []string{"integration_test_2"},
		},
	}

	updatedComponent, apiCmd := component.Update(completeMsg)
	component = updatedComponent.(*TestComponent)

	// Execute the API update command
	if apiCmd != nil {
		apiUpdateMsg := apiCmd()
		updatedComponent, _ = component.Update(apiUpdateMsg)
		component = updatedComponent.(*TestComponent)
	}

	// Verify test completed state
	if component.testing {
		t.Error("Expected testing to be false after completion")
	}

	if !component.showingTestResults {
		t.Error("Expected showingTestResults to be true after completion")
	}

	// Simulate API update completion
	apiMsg := apiUpdateCompleteMsg{err: nil}
	updatedComponent, _ = component.Update(apiMsg)
	component = updatedComponent.(*TestComponent)

	if !apiCallMade {
		t.Error("Expected API call to be made")
	}

	if !strings.Contains(component.testResultsSummary, "[API update successful!]") {
		t.Error("Expected success message in summary")
	}
}
