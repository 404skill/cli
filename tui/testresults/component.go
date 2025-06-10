package testresults

import (
	"fmt"
	"strings"

	"404skill-cli/testreport"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles for the test results component
var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00ffaa")).
			Underline(true).
			Padding(0, 1)

	passedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00aa00"))

	failedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ff0000"))

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#00aa00")).
			Foreground(lipgloss.Color("#000000")).
			Bold(true)

	expandedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Italic(true).
			Padding(0, 1)

	failureContentStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#ff0000")).
				Padding(0, 1).
				MarginLeft(0)

	outputStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1a1a1a")).
			Foreground(lipgloss.Color("#cccccc")).
			Padding(0, 1).
			MarginLeft(0)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Faint(true)
)

// TestResultsComponent handles the expandable test results display
type TestResultsComponent struct {
	// Dependencies
	help help.Model

	// State
	results           *testreport.ParseResult
	items             []TestResultItem
	selectedIndex     int
	lastSelectedIndex int
	expandedTests     map[string]bool
	activeSection     FailureSection
}

// Key bindings
type keyMap struct {
	Up          key.Binding
	Down        key.Binding
	Expand      key.Binding
	Collapse    key.Binding
	Toggle      key.Binding
	NextSection key.Binding
	PageUp      key.Binding
	PageDown    key.Binding
	ScrollUp    key.Binding
	ScrollDown  key.Binding
	Back        key.Binding
	Quit        key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Expand: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "expand"),
	),
	Collapse: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "collapse"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("enter/space", "toggle"),
	),
	NextSection: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next section"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "page up"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "page down"),
	),
	ScrollUp: key.NewBinding(
		key.WithKeys("ctrl+k", "shift+up"),
		key.WithHelp("ctrl+k", "scroll up"),
	),
	ScrollDown: key.NewBinding(
		key.WithKeys("ctrl+j", "shift+down"),
		key.WithHelp("ctrl+j", "scroll down"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "b"),
		key.WithHelp("esc/b", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

// New creates a new test results component
func New() *TestResultsComponent {
	return &TestResultsComponent{
		help:          help.New(),
		expandedTests: make(map[string]bool),
		activeSection: SectionMessage,
	}
}

// Init initializes the component
func (c *TestResultsComponent) Init() tea.Cmd {
	return nil
}

// SetResults sets the test results and builds the display items
func (c *TestResultsComponent) SetResults(results *testreport.ParseResult) {
	c.results = results
	c.buildItems()
}

// GetSelectedTest returns the currently selected test result
func (c *TestResultsComponent) GetSelectedTest() *testreport.TestResult {
	if c.selectedIndex >= 0 && c.selectedIndex < len(c.items) {
		return &c.items[c.selectedIndex].Result
	}
	return nil
}

// Update handles incoming messages
func (c *TestResultsComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Handle window size change

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Up):
			if c.selectedIndex > 0 {
				c.selectedIndex--
				// Reset scroll when navigating to different test
				if c.selectedIndex != c.lastSelectedIndex {
					c.lastSelectedIndex = c.selectedIndex
				}
			}

		case key.Matches(msg, keys.Down):
			if c.selectedIndex < len(c.items)-1 {
				c.selectedIndex++
				// Reset scroll when navigating to different test
				if c.selectedIndex != c.lastSelectedIndex {
					c.lastSelectedIndex = c.selectedIndex
				}
			}

		case key.Matches(msg, keys.Expand):
			if c.selectedIndex >= 0 && c.selectedIndex < len(c.items) {
				item := &c.items[c.selectedIndex]
				if !item.Result.Passed {
					c.expandedTests[item.Result.Name] = true
				}
			}

		case key.Matches(msg, keys.Collapse):
			if c.selectedIndex >= 0 && c.selectedIndex < len(c.items) {
				item := &c.items[c.selectedIndex]
				c.expandedTests[item.Result.Name] = false
			}

		case key.Matches(msg, keys.Toggle):
			if c.selectedIndex >= 0 && c.selectedIndex < len(c.items) {
				item := &c.items[c.selectedIndex]
				if !item.Result.Passed {
					current := c.expandedTests[item.Result.Name]
					c.expandedTests[item.Result.Name] = !current
				}
			}

		case key.Matches(msg, keys.NextSection):
			c.activeSection = (c.activeSection + 1) % 3

		case key.Matches(msg, keys.PageUp):
			// Debug: Add some visual feedback when scrolling
			return c, nil

		case key.Matches(msg, keys.PageDown):
			// Debug: Add some visual feedback when scrolling
			return c, nil

		case key.Matches(msg, keys.ScrollUp):
			return c, nil

		case key.Matches(msg, keys.ScrollDown):
			return c, nil

		case key.Matches(msg, keys.Back):
			return c, func() tea.Msg { return BackToTestListMsg{} }

		case key.Matches(msg, keys.Quit):
			return c, tea.Quit
		}
	}

	return c, nil
}

// View renders the component
func (c *TestResultsComponent) View() string {
	if c.results == nil {
		return "No test results available"
	}

	// Ensure content is always up to date
	c.buildItems()

	// Header with summary
	header := c.buildHeaderView()

	// Help with scroll indicators
	helpView := helpStyle.Render(c.help.View(keys))

	// Main content
	content := c.buildTestListView()

	return fmt.Sprintf("%s\n\n%s\n\n%s", header, content, helpView)
}

// buildItems creates the list of test result items
func (c *TestResultsComponent) buildItems() {
	if c.results == nil {
		return
	}

	c.items = make([]TestResultItem, len(c.results.Suite.Results))
	for i, result := range c.results.Suite.Results {
		c.items[i] = TestResultItem{
			Result:   result,
			Expanded: c.expandedTests[result.Name],
			Selected: i == c.selectedIndex,
		}
	}
}

// buildHeaderView creates the summary header
func (c *TestResultsComponent) buildHeaderView() string {
	if c.results == nil {
		return ""
	}

	suite := c.results.Suite
	testCount := suite.Tests
	passedCount := len(c.results.PassedTests)
	failedCount := len(c.results.FailedTests)
	testTime := suite.Time

	summary := fmt.Sprintf(
		"Total: %d   Passed: %d   Failed: %d   Time: %.2fs",
		testCount, passedCount, failedCount, testTime,
	)

	return fmt.Sprintf("%s\n%s",
		headerStyle.Render("Test Results: "+suite.Name),
		summary)
}

// buildTestListView creates the main test list view
func (c *TestResultsComponent) buildTestListView() string {
	var b strings.Builder
	for _, item := range c.items {
		// Select style
		line := c.formatTestLine(item)
		if item.Selected {
			line = selectedStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")

		// Only show the first line of output or failure message if expanded
		if item.Expanded {
			var detail string
			if item.Result.Passed {
				if item.Result.Output != nil && len(item.Result.Output.Stdout) > 0 {
					detail = strings.SplitN(item.Result.Output.Stdout, "\n", 2)[0]
				}
				if detail != "" {
					b.WriteString(passedStyle.Render(detail) + "\n")
				}
			} else if item.Result.Failure != nil {
				msg := item.Result.Failure.Message
				if msg == "" && item.Result.Output != nil && len(item.Result.Output.Stdout) > 0 {
					msg = strings.SplitN(item.Result.Output.Stdout, "\n", 2)[0]
				} else if msg != "" {
					msg = strings.SplitN(msg, "\n", 2)[0]
				}
				if msg != "" {
					b.WriteString(failedStyle.Render(msg) + "\n")
				}
			}
		}
	}
	return b.String()
}

// formatTestLine formats a single test result line
func (c *TestResultsComponent) formatTestLine(item TestResultItem) string {
	result := item.Result
	status := ""
	expansion := ""

	if result.Passed {
		status = passedStyle.Render("[PASS]")
	} else {
		status = failedStyle.Render("[FAIL]")
		if item.Expanded {
			expansion = " [-]"
		} else {
			expansion = " [+]"
		}
	}

	return fmt.Sprintf("%s  %s%s  (%.2fs)",
		status, result.Name, expansion, result.Time)
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Toggle, k.PageUp, k.PageDown, k.Back}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Expand, k.Collapse, k.Toggle},
		{k.PageUp, k.PageDown, k.ScrollUp, k.ScrollDown},
		{k.NextSection, k.Back, k.Quit},
	}
}
