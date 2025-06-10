package testresults

import (
	"strings"
	"testing"

	"404skill-cli/testreport"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNew(t *testing.T) {
	component := New()

	if component == nil {
		t.Fatal("Expected component to be created")
	}

	if component.expandedTests == nil {
		t.Error("Expected expandedTests map to be initialized")
	}

	if component.activeSection != SectionMessage {
		t.Error("Expected activeSection to be SectionMessage initially")
	}

	if component.selectedIndex != 0 {
		t.Error("Expected selectedIndex to be 0 initially")
	}
}

func TestSetResults(t *testing.T) {
	component := New()

	// Create test results
	results := &testreport.ParseResult{
		Suite: testreport.TestSuite{
			Name:  "Test Suite",
			Tests: 3,
			Time:  1.5,
		},
		PassedTests: []string{"test1", "test2"},
		FailedTests: []string{"test3"},
	}
	results.Suite.Results = []testreport.TestResult{
		{Name: "test1", Passed: true, Time: 0.5},
		{Name: "test2", Passed: true, Time: 0.3},
		{Name: "test3", Passed: false, Time: 0.7, Failure: &testreport.TestFailure{
			Message: "Test failed",
			Type:    "AssertionError",
			Content: "Expected 1 but got 2",
		}},
	}

	component.SetResults(results)

	if component.results != results {
		t.Error("Expected results to be set")
	}

	if len(component.items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(component.items))
	}

	// Verify items are correctly built
	for i, item := range component.items {
		if item.Result.Name != results.Suite.Results[i].Name {
			t.Errorf("Expected item %d name to be %s, got %s", i, results.Suite.Results[i].Name, item.Result.Name)
		}
	}
}

func TestGetSelectedTest(t *testing.T) {
	component := New()

	// Test with no results
	selected := component.GetSelectedTest()
	if selected != nil {
		t.Error("Expected nil when no results are set")
	}

	// Set up test results
	results := &testreport.ParseResult{
		Suite: testreport.TestSuite{
			Name: "Test Suite",
		},
	}
	results.Suite.Results = []testreport.TestResult{
		{Name: "test1", Passed: true, Time: 0.5},
		{Name: "test2", Passed: false, Time: 0.3},
	}

	component.SetResults(results)

	// Test getting selected test
	selected = component.GetSelectedTest()
	if selected == nil {
		t.Error("Expected to get selected test")
	}

	if selected.Name != "test1" {
		t.Errorf("Expected selected test to be 'test1', got '%s'", selected.Name)
	}

	// Change selection
	component.selectedIndex = 1
	selected = component.GetSelectedTest()
	if selected.Name != "test2" {
		t.Errorf("Expected selected test to be 'test2', got '%s'", selected.Name)
	}
}

func TestUpdate_Navigation(t *testing.T) {
	component := New()

	// Set up test results
	results := &testreport.ParseResult{
		Suite: testreport.TestSuite{Name: "Test Suite"},
	}
	results.Suite.Results = []testreport.TestResult{
		{Name: "test1", Passed: true, Time: 0.5},
		{Name: "test2", Passed: false, Time: 0.3},
		{Name: "test3", Passed: true, Time: 0.2},
	}
	component.SetResults(results)

	tests := []struct {
		name          string
		keyMsg        string
		expectedIndex int
		initialIndex  int
	}{
		{
			name:          "down key moves selection down",
			keyMsg:        "down",
			expectedIndex: 1,
			initialIndex:  0,
		},
		{
			name:          "up key moves selection up",
			keyMsg:        "up",
			expectedIndex: 0,
			initialIndex:  1,
		},
		{
			name:          "down key at end does nothing",
			keyMsg:        "down",
			expectedIndex: 2,
			initialIndex:  2,
		},
		{
			name:          "up key at start does nothing",
			keyMsg:        "up",
			expectedIndex: 0,
			initialIndex:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component.selectedIndex = tt.initialIndex

			keyMsg := tea.KeyMsg{
				Type:  tea.KeyRunes,
				Runes: []rune(tt.keyMsg),
			}

			updatedComponent, _ := component.Update(keyMsg)
			component = updatedComponent.(*TestResultsComponent)

			if component.selectedIndex != tt.expectedIndex {
				t.Errorf("Expected selectedIndex to be %d, got %d", tt.expectedIndex, component.selectedIndex)
			}
		})
	}
}

func TestUpdate_Expansion(t *testing.T) {
	component := New()

	// Set up test results with failed test
	results := &testreport.ParseResult{
		Suite: testreport.TestSuite{Name: "Test Suite"},
	}
	results.Suite.Results = []testreport.TestResult{
		{Name: "test1", Passed: true, Time: 0.5},
		{Name: "failed_test", Passed: false, Time: 0.3, Failure: &testreport.TestFailure{
			Message: "Test failed",
		}},
	}
	component.SetResults(results)

	// Select the failed test
	component.selectedIndex = 1

	// Test expand
	expandMsg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("right"),
	}

	updatedComponent, _ := component.Update(expandMsg)
	component = updatedComponent.(*TestResultsComponent)

	if !component.expandedTests["failed_test"] {
		t.Error("Expected failed test to be expanded")
	}

	// Test collapse
	collapseMsg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("left"),
	}

	updatedComponent, _ = component.Update(collapseMsg)
	component = updatedComponent.(*TestResultsComponent)

	if component.expandedTests["failed_test"] {
		t.Error("Expected failed test to be collapsed")
	}

	// Test toggle
	toggleMsg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("enter"),
	}

	updatedComponent, _ = component.Update(toggleMsg)
	component = updatedComponent.(*TestResultsComponent)

	if !component.expandedTests["failed_test"] {
		t.Error("Expected failed test to be expanded after toggle")
	}
}

func TestUpdate_BackMessage(t *testing.T) {
	component := New()

	backMsg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("esc"),
	}

	_, cmd := component.Update(backMsg)

	if cmd == nil {
		t.Error("Expected command to be returned for back message")
	}

	// Execute the command to get the message
	msg := cmd()
	if _, ok := msg.(BackToTestListMsg); !ok {
		t.Error("Expected BackToTestListMsg")
	}
}

func TestView_NoResults(t *testing.T) {
	component := New()

	view := component.View()

	if !strings.Contains(view, "No test results available") {
		t.Error("Expected 'No test results available' message")
	}
}

func TestView_WithResults(t *testing.T) {
	component := New()

	// Set up test results
	results := &testreport.ParseResult{
		Suite: testreport.TestSuite{
			Name:  "Test Suite",
			Tests: 2,
			Time:  1.0,
		},
		PassedTests: []string{"test1"},
		FailedTests: []string{"test2"},
	}
	results.Suite.Results = []testreport.TestResult{
		{Name: "test1", Passed: true, Time: 0.5},
		{Name: "test2", Passed: false, Time: 0.5, Failure: &testreport.TestFailure{
			Message: "Test failed",
			Content: "Assertion error details",
		}},
	}

	component.SetResults(results)

	view := component.View()

	// Check header content
	if !strings.Contains(view, "Test Results: Test Suite") {
		t.Error("Expected header to contain suite name")
	}

	if !strings.Contains(view, "Total: 2") {
		t.Error("Expected total count in header")
	}

	if !strings.Contains(view, "Passed: 1") {
		t.Error("Expected passed count in header")
	}

	if !strings.Contains(view, "Failed: 1") {
		t.Error("Expected failed count in header")
	}

	// Check test list content
	if !strings.Contains(view, "[PASS]") {
		t.Error("Expected [PASS] marker")
	}

	if !strings.Contains(view, "[FAIL]") {
		t.Error("Expected [FAIL] marker")
	}

	if !strings.Contains(view, "test1") {
		t.Error("Expected test1 in view")
	}

	if !strings.Contains(view, "test2") {
		t.Error("Expected test2 in view")
	}
}

func TestView_ExpandedFailure(t *testing.T) {
	component := New()

	// Set up test results with failed test
	results := &testreport.ParseResult{
		Suite: testreport.TestSuite{Name: "Test Suite"},
	}
	results.Suite.Results = []testreport.TestResult{
		{Name: "failed_test", Passed: false, Time: 0.5, Failure: &testreport.TestFailure{
			Message: "Assertion failed",
			Content: "Expected true but got false",
		}},
	}

	component.SetResults(results)

	// Expand the failed test
	component.expandedTests["failed_test"] = true

	view := component.View()

	// Check that failure details are shown
	if !strings.Contains(view, "⚠  Assertion failed") {
		t.Error("Expected failure message to be shown when expanded")
	}

	if !strings.Contains(view, "Expected true but got false") {
		t.Error("Expected failure content to be shown when expanded")
	}
}

func TestFormatTestLine(t *testing.T) {
	component := New()

	tests := []struct {
		name              string
		item              TestResultItem
		expectedStatus    string
		expectedExpansion string
	}{
		{
			name: "passed test",
			item: TestResultItem{
				Result: testreport.TestResult{
					Name:   "passing_test",
					Passed: true,
					Time:   0.5,
				},
			},
			expectedStatus:    "[PASS]",
			expectedExpansion: "",
		},
		{
			name: "failed test collapsed",
			item: TestResultItem{
				Result: testreport.TestResult{
					Name:   "failing_test",
					Passed: false,
					Time:   0.8,
				},
				Expanded: false,
			},
			expectedStatus:    "[FAIL]",
			expectedExpansion: "[+]",
		},
		{
			name: "failed test expanded",
			item: TestResultItem{
				Result: testreport.TestResult{
					Name:   "failing_test",
					Passed: false,
					Time:   0.8,
				},
				Expanded: true,
			},
			expectedStatus:    "[FAIL]",
			expectedExpansion: "[-]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line := component.formatTestLine(tt.item)

			if !strings.Contains(line, tt.expectedStatus) {
				t.Errorf("Expected line to contain %s, got: %s", tt.expectedStatus, line)
			}

			if tt.expectedExpansion != "" && !strings.Contains(line, tt.expectedExpansion) {
				t.Errorf("Expected line to contain %s, got: %s", tt.expectedExpansion, line)
			}

			if !strings.Contains(line, tt.item.Result.Name) {
				t.Errorf("Expected line to contain test name %s, got: %s", tt.item.Result.Name, line)
			}
		})
	}
}

func TestInit(t *testing.T) {
	component := New()

	cmd := component.Init()

	if cmd != nil {
		t.Error("Expected Init to return nil command")
	}
}
