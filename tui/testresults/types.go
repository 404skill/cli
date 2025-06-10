package testresults

import (
	"404skill-cli/testreport"

	tea "github.com/charmbracelet/bubbletea"
)

// Component interface for the test results viewer
type Component interface {
	Init() tea.Cmd
	Update(tea.Msg) (tea.Model, tea.Cmd)
	View() string
	SetResults(*testreport.ParseResult)
	GetSelectedTest() *testreport.TestResult
}

// ToggleExpansionMsg is sent when user wants to expand/collapse a failed test
type ToggleExpansionMsg struct {
	TestName string
}

// BackToTestListMsg is sent when user wants to return to test list
type BackToTestListMsg struct{}

// NavigateToSectionMsg is sent when user navigates between failure sections
type NavigateToSectionMsg struct {
	Section FailureSection
}

// FailureSection represents different sections of a test failure
type FailureSection int

const (
	SectionMessage FailureSection = iota
	SectionStdout
	SectionStderr
)

// TestResultItem represents a test result in the list with UI state
type TestResultItem struct {
	Result   testreport.TestResult
	Expanded bool
	Selected bool
}
