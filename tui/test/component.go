package test

import (
	"context"
	"fmt"
	"strings"
	"time"

	"404skill-cli/api"
	"404skill-cli/testreport"
	"404skill-cli/testrunner"
	"404skill-cli/tracing"
	"404skill-cli/tui/testresults"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	btable "github.com/evertras/bubble-table/table"
)

var (
	// Styles
	headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	successStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))

	// Spinner frames for animation
	spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
)

// Component handles the test project UI
type TestComponent struct {
	// Dependencies
	testRunner    testrunner.TestRunner
	configManager ConfigManager
	apiClient     APIClient

	// UI State
	table                btable.Model
	help                 help.Model
	spinnerFrame         string
	showingTestResults   bool
	testResultsComponent *testresults.TestResultsComponent

	// Data
	projects           []testrunner.Project
	currentProject     *testrunner.Project
	testResultsSummary string
	testResultsList    []string

	// State
	testing      bool
	errorMsg     string
	outputBuffer []string
}

// New creates a new TestComponent with dependency injection
func New(testRunner testrunner.TestRunner, configManager ConfigManager, apiClient APIClient) *TestComponent {
	columns := []btable.Column{
		btable.NewColumn("name", "Project", 40),
		btable.NewColumn("lang", "Language", 15),
		btable.NewColumn("diff", "Difficulty", 15),
		btable.NewColumn("dur", "Duration", 15),
		btable.NewColumn("status", "Status", 20),
	}

	table := btable.New(columns).WithRows([]btable.Row{}).Focused(true)

	return &TestComponent{
		testRunner:    testRunner,
		configManager: configManager,
		apiClient:     apiClient,
		table:         table,
		help:          help.New(),
		spinnerFrame:  spinnerFrames[0],
	}
}

// Init initializes the component
func (c *TestComponent) Init() tea.Cmd {
	return nil
}

// SetProjects updates the list of projects and rebuilds the table
func (c *TestComponent) SetProjects(projects []api.Project) {
	c.projects = nil
	rows := []btable.Row{}

	for _, p := range projects {
		if c.configManager.IsProjectDownloaded(p.ID) {
			project := testrunner.Project{
				ID:       p.ID,
				Name:     p.Name,
				Language: p.Language,
			}
			c.projects = append(c.projects, project)

			rows = append(rows, btable.NewRow(map[string]interface{}{
				"id":     p.ID,
				"name":   p.Name,
				"lang":   p.Language,
				"diff":   p.Difficulty,
				"dur":    fmt.Sprintf("%d min", p.EstimatedDurationInMinutes),
				"status": "✓ Downloaded",
			}))
		}
	}

	c.table = c.table.WithRows(rows)
}

// Update handles incoming messages
func (c *TestComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if c.showingTestResults {
			// Handle dismissing test results
			switch msg.String() {
			case "esc", "b":
				c.showingTestResults = false
				c.testResultsComponent = nil
				c.testResultsSummary = ""
				c.testResultsList = nil
				return c, nil
			default:
				// Delegate to testresults component if it exists
				if c.testResultsComponent != nil {
					updatedComponent, cmd := c.testResultsComponent.Update(msg)
					c.testResultsComponent = updatedComponent.(*testresults.TestResultsComponent)

					// Check for back message
					if cmd != nil {
						if backMsg := cmd(); backMsg != nil {
							if _, ok := backMsg.(testresults.BackToTestListMsg); ok {
								c.showingTestResults = false
								c.testResultsComponent = nil
								c.testResultsSummary = ""
								c.testResultsList = nil
								return c, nil
							}
						}
					}
					return c, cmd
				}
				return c, nil
			}
		}

		if c.testing {
			// Don't handle input while testing
			return c, nil
		}

		switch msg.String() {
		case "enter":
			selected := c.table.HighlightedRow()
			if selected.Data != nil {
				if id, ok := selected.Data["id"].(string); ok {
					for _, p := range c.projects {
						if p.ID == id {
							// Clear ALL previous test state
							c.showingTestResults = false
							c.testResultsComponent = nil
							c.testResultsSummary = ""
							c.testResultsList = nil
							c.errorMsg = ""
							c.outputBuffer = nil
							c.currentProject = nil

							c.testing = true
							c.currentProject = &p
							return c, tea.Batch(
								c.runTestsCmd(p),
								c.spinnerTick(),
							)
						}
					}
				}
			}
		case "esc", "b":
			// If we're not showing test results, let the parent handle back navigation
			if !c.showingTestResults {
				return c, nil
			}
		}

	case TestCompleteMsg:
		c.testing = false
		if msg.Error != "" {
			_ = tracing.TrackError(fmt.Errorf("test completed with error: %s", msg.Error), "test_component")
			c.errorMsg = msg.Error
			return c, nil
		}

		// Show test results
		c.showingTestResults = true
		c.buildTestResultsView(msg.Result)

		// Update API - use project from message instead of component state
		return c, c.updateAPICmd(msg.Result, msg.Project)

	case TestProgressMsg:
		if msg.Line != "" {
			c.outputBuffer = append(c.outputBuffer, msg.Line)
		}
		return c, nil

	case TestErrorMsg:
		c.testing = false
		c.errorMsg = msg.Error
		return c, nil

	case spinnerMsg:
		c.spinnerFrame = msg.frame
		if c.testing {
			return c, c.spinnerTick()
		}
		return c, nil

	case apiUpdateCompleteMsg:
		if msg.err != nil {
			c.testResultsSummary += "\n\n[API update failed: " + msg.err.Error() + "]"
		} else {
			c.testResultsSummary += "\n\n[API update successful!]"
		}
		return c, nil
	}

	c.table, cmd = c.table.Update(msg)
	return c, cmd
}

// View renders the component
func (c *TestComponent) View() string {
	if c.showingTestResults {
		if c.testResultsComponent != nil {
			// Use the enhanced test results component
			return c.testResultsComponent.View()
		}
		// Fallback to original view if component not available
		var b strings.Builder
		b.WriteString(c.testResultsSummary)
		b.WriteString("\n\n")
		for _, line := range c.testResultsList {
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("\nPress any key to return to the project list.")
		return b.String()
	}

	if c.testing {
		out := strings.Join(c.outputBuffer, "\n")
		return fmt.Sprintf("%s\n\nRunning tests...\n%s\n%s\n\nPress q to quit",
			headerStyle.Render("Testing Project"),
			spinnerStyle.Render(c.spinnerFrame),
			out)
	}

	// Show project table
	keyMap := struct {
		Enter, Back, Quit string
	}{
		Enter: "enter",
		Back:  "esc/b",
		Quit:  "q",
	}

	helpView := helpStyle.Render(fmt.Sprintf("[%s] select • [%s] back • [%s] quit",
		keyMap.Enter, keyMap.Back, keyMap.Quit))
	view := fmt.Sprintf("%s\n%s", c.table.View(), helpView)

	if c.errorMsg != "" {
		view = fmt.Sprintf("%s\n\n%s", view, errorStyle.Render(c.errorMsg))
	}

	return view
}

// buildTestResultsView constructs the test results display
func (c *TestComponent) buildTestResultsView(result *testreport.ParseResult) {
	// Create and configure the enhanced test results component
	c.testResultsComponent = testresults.New()
	c.testResultsComponent.SetResults(result)

	// Keep the original summary for API update messages
	testCount := result.Suite.Tests
	passedCount := len(result.PassedTests)
	failedCount := len(result.FailedTests)
	testTime := result.Suite.Time

	c.testResultsSummary = fmt.Sprintf(
		"%s\n\nTotal: %d   Passed: %d   Failed: %d   Time: %.2fs",
		headerStyle.Render("Test Results: "+result.Suite.Name),
		testCount, passedCount, failedCount, testTime,
	)
}

// runTestsCmd creates a command to run tests for a project
func (c *TestComponent) runTestsCmd(project testrunner.Project) tea.Cmd {
	return func() tea.Msg {
		progressCallback := func(line string) {
			// Progress callback - could be enhanced to send real-time updates
			// For now, the enhanced error messages will contain full output
		}

		result, err := c.testRunner.RunTests(project, progressCallback)
		if err != nil {
			return TestCompleteMsg{
				Project: &project,
				Error:   err.Error(),
			}
		}

		return TestCompleteMsg{
			Project: &project,
			Result:  result,
		}
	}
}

// updateAPICmd creates a command to update the API with test results
func (c *TestComponent) updateAPICmd(result *testreport.ParseResult, project *testrunner.Project) tea.Cmd {
	return func() tea.Msg {
		tracker := tracing.TimedOperation("api_bulk_update_profile_tests")

		if project == nil {
			_ = tracker.CompleteWithError(fmt.Errorf("no current project"))
			return apiUpdateCompleteMsg{err: fmt.Errorf("no current project")}
		}

		tracker.AddMetadata("project_id", project.ID)
		tracker.AddMetadata("passed_count", fmt.Sprintf("%d", len(result.PassedTests)))
		tracker.AddMetadata("failed_count", fmt.Sprintf("%d", len(result.FailedTests)))

		ctx := context.Background()
		err := c.apiClient.BulkUpdateProfileTests(
			ctx,
			result.FailedTests,
			result.PassedTests,
			project.ID,
		)

		if err != nil {
			_ = tracker.CompleteWithError(err)
		} else {
			_ = tracker.Complete()
		}

		return apiUpdateCompleteMsg{err: err}
	}
}

// Spinner animation message and command
type spinnerMsg struct{ frame string }

func (c *TestComponent) spinnerTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
		idx := 0
		for i, f := range spinnerFrames {
			if f == c.spinnerFrame {
				idx = i
				break
			}
		}
		return spinnerMsg{spinnerFrames[(idx+1)%len(spinnerFrames)]}
	})
}

// API update completion message
type apiUpdateCompleteMsg struct{ err error }

// IsShowingTestResults returns whether test results are currently being displayed
func (c *TestComponent) IsShowingTestResults() bool {
	return c.showingTestResults
}
