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

	groupHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#ffaa00")).
				Background(lipgloss.Color("#2a2a2a")).
				Padding(0, 1).
				MarginTop(1)

	groupDividerStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#444444")).
				Bold(true)

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

// DisplayItemType represents the type of display item
type DisplayItemType int

const (
	ItemTypeGroupHeader DisplayItemType = iota
	ItemTypeTest
	ItemTypeDivider
)

// DisplayItem represents an item in the display list (group header, test, or divider)
type DisplayItem struct {
	Type     DisplayItemType
	Test     *TestResultItem  // For test items
	Group    *GroupHeaderItem // For group headers
	Selected bool
}

// GroupHeaderItem represents a group header display item
type GroupHeaderItem struct {
	Name        string
	DisplayName string
	PassedCount int
	FailedCount int
	TotalTime   float64
}

// TestResultsComponent handles the expandable test results display
type TestResultsComponent struct {
	// Dependencies
	help help.Model

	// State
	results           *testreport.ParseResult
	items             []TestResultItem // Legacy: individual tests
	displayItems      []DisplayItem    // New: grouped display with headers
	selectedIndex     int
	lastSelectedIndex int
	expandedTests     map[string]bool
	activeSection     FailureSection

	// Scrolling
	visibleStart int // index of first visible item
	listHeight   int // number of lines available for the list
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
	// Ensure selection is on a test item
	c.ensureValidSelection()
}

// ensureValidSelection ensures the selection is on a test item, not a header or divider
func (c *TestResultsComponent) ensureValidSelection() {
	if len(c.displayItems) == 0 {
		c.selectedIndex = 0
		return
	}

	// If current selection is valid, keep it
	if c.selectedIndex >= 0 && c.selectedIndex < len(c.displayItems) {
		if c.displayItems[c.selectedIndex].Type == ItemTypeTest {
			return
		}
	}

	// Find the first test item
	for i, item := range c.displayItems {
		if item.Type == ItemTypeTest {
			c.selectedIndex = i
			c.buildItems() // Rebuild to update selection state
			return
		}
	}

	// No test items found
	c.selectedIndex = 0
}

// GetSelectedTest returns the currently selected test result
func (c *TestResultsComponent) GetSelectedTest() *testreport.TestResult {
	if c.selectedIndex >= 0 && c.selectedIndex < len(c.displayItems) {
		item := c.displayItems[c.selectedIndex]
		if item.Type == ItemTypeTest && item.Test != nil {
			return &item.Test.Result
		}
	}
	return nil
}

// Update handles incoming messages
func (c *TestResultsComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Reserve 4 lines: header (2), help (1), padding (1)
		c.listHeight = msg.Height - 4
		if c.listHeight < 1 {
			c.listHeight = 1
		}
		// Clamp visibleStart if needed
		if c.visibleStart > len(c.items)-c.listHeight {
			c.visibleStart = max(0, len(c.items)-c.listHeight)
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Up):
			c.navigateUp()

		case key.Matches(msg, keys.Down):
			c.navigateDown()

		case key.Matches(msg, keys.Expand):
			if c.selectedIndex >= 0 && c.selectedIndex < len(c.displayItems) {
				item := c.displayItems[c.selectedIndex]
				if item.Type == ItemTypeTest && item.Test != nil && !item.Test.Result.Passed {
					c.expandedTests[item.Test.Result.Name] = true
					c.buildItems()
				}
			}

		case key.Matches(msg, keys.Collapse):
			if c.selectedIndex >= 0 && c.selectedIndex < len(c.displayItems) {
				item := c.displayItems[c.selectedIndex]
				if item.Type == ItemTypeTest && item.Test != nil {
					c.expandedTests[item.Test.Result.Name] = false
					c.buildItems()
				}
			}

		case key.Matches(msg, keys.Toggle):
			if c.selectedIndex >= 0 && c.selectedIndex < len(c.displayItems) {
				item := c.displayItems[c.selectedIndex]
				if item.Type == ItemTypeTest && item.Test != nil && !item.Test.Result.Passed {
					current := c.expandedTests[item.Test.Result.Name]
					c.expandedTests[item.Test.Result.Name] = !current
					c.buildItems()
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

	// Build legacy items for compatibility
	c.items = make([]TestResultItem, len(c.results.Suite.Results))
	for i, result := range c.results.Suite.Results {
		c.items[i] = TestResultItem{
			Result:   result,
			Expanded: c.expandedTests[result.Name],
			Selected: false, // Selection handled in displayItems
		}
	}

	// Build grouped display items
	c.displayItems = []DisplayItem{}

	if c.results.GroupedResults != nil {
		// Use grouped results
		for groupIndex, group := range c.results.GroupedResults.Classes {
			// Add group header
			header := DisplayItem{
				Type: ItemTypeGroupHeader,
				Group: &GroupHeaderItem{
					Name:        group.Name,
					DisplayName: group.DisplayName,
					PassedCount: group.PassedCount,
					FailedCount: group.FailedCount,
					TotalTime:   group.TotalTime,
				},
				Selected: false, // Headers are not selectable
			}
			c.displayItems = append(c.displayItems, header)

			// Add tests for this group
			for _, test := range group.Tests {
				testItem := DisplayItem{
					Type: ItemTypeTest,
					Test: &TestResultItem{
						Result:   test,
						Expanded: c.expandedTests[test.Name],
					},
					Selected: false, // Will be set below
				}
				c.displayItems = append(c.displayItems, testItem)
			}

			// Add divider between groups (except after last group)
			if groupIndex < len(c.results.GroupedResults.Classes)-1 {
				divider := DisplayItem{
					Type:     ItemTypeDivider,
					Selected: false, // Dividers are not selectable
				}
				c.displayItems = append(c.displayItems, divider)
			}
		}
	} else {
		// Fallback: use original results without grouping
		for _, result := range c.results.Suite.Results {
			testItem := DisplayItem{
				Type: ItemTypeTest,
				Test: &TestResultItem{
					Result:   result,
					Expanded: c.expandedTests[result.Name],
				},
				Selected: false,
			}
			c.displayItems = append(c.displayItems, testItem)
		}
	}

	// Update selection state - only for test items
	for i := range c.displayItems {
		if c.displayItems[i].Type == ItemTypeTest && i == c.selectedIndex {
			c.displayItems[i].Selected = true
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
	if c.listHeight <= 0 {
		c.listHeight = 10 // fallback default
	}
	start := c.visibleStart
	end := min(start+c.listHeight, len(c.displayItems))
	var b strings.Builder

	for i := start; i < end; i++ {
		item := c.displayItems[i]

		switch item.Type {
		case ItemTypeGroupHeader:
			line := c.formatGroupHeader(item)
			if item.Selected {
				line = selectedStyle.Render(line)
			}
			b.WriteString(line)
			b.WriteString("\n")

		case ItemTypeTest:
			if item.Test != nil {
				line := c.formatTestLine(*item.Test)
				if item.Selected {
					line = selectedStyle.Render(line)
				}
				b.WriteString(line)
				b.WriteString("\n")

				// Show failure message if expanded
				if item.Test.Expanded {
					var detail string
					if item.Test.Result.Passed {
						if item.Test.Result.Output != nil && len(item.Test.Result.Output.Stdout) > 0 {
							detail = strings.SplitN(item.Test.Result.Output.Stdout, "\n", 2)[0]
						}
						if detail != "" {
							b.WriteString(passedStyle.Render("  "+detail) + "\n")
						}
					} else if item.Test.Result.Failure != nil {
						msg := item.Test.Result.Failure.Message
						if msg == "" && item.Test.Result.Output != nil && len(item.Test.Result.Output.Stdout) > 0 {
							msg = strings.SplitN(item.Test.Result.Output.Stdout, "\n", 2)[0]
						} else if msg != "" {
							msg = strings.SplitN(msg, "\n", 2)[0]
						}
						if msg != "" {
							b.WriteString(failedStyle.Render("  "+msg) + "\n")
						}
					}
				}
			}

		case ItemTypeDivider:
			dividerLine := groupDividerStyle.Render("────────────────────────────────────────")
			b.WriteString(dividerLine)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// formatGroupHeader formats a group header line
func (c *TestResultsComponent) formatGroupHeader(item DisplayItem) string {
	if item.Group == nil {
		return ""
	}

	group := item.Group
	header := groupHeaderStyle.Render(fmt.Sprintf("📁 %s", group.DisplayName))

	// Add statistics
	stats := fmt.Sprintf("(%d passed, %d failed, %.2fs)",
		group.PassedCount, group.FailedCount, group.TotalTime)

	return fmt.Sprintf("%s %s", header, stats)
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
	return []key.Binding{k.Up, k.Down, k.Toggle, k.Back, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Expand, k.Collapse, k.Toggle},
		{k.NextSection, k.Back, k.Quit},
	}
}

// Utility functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (c *TestResultsComponent) navigateUp() {
	originalIndex := c.selectedIndex

	if c.selectedIndex > 0 {
		c.selectedIndex--

		// Skip non-selectable items
		for c.selectedIndex >= 0 && c.selectedIndex < len(c.displayItems) {
			if c.displayItems[c.selectedIndex].Type == ItemTypeTest {
				break // Found a selectable test item
			}
			if c.selectedIndex > 0 {
				c.selectedIndex--
			} else {
				// Can't go further up, revert
				c.selectedIndex = originalIndex
				return
			}
		}

		// Update view and rebuild
		if c.selectedIndex < c.visibleStart {
			c.visibleStart = c.selectedIndex
		}
		if c.selectedIndex != c.lastSelectedIndex {
			c.lastSelectedIndex = c.selectedIndex
		}
		c.buildItems()
	}
}

func (c *TestResultsComponent) navigateDown() {
	originalIndex := c.selectedIndex

	if c.selectedIndex < len(c.displayItems)-1 {
		c.selectedIndex++

		// Skip non-selectable items
		for c.selectedIndex < len(c.displayItems) {
			if c.displayItems[c.selectedIndex].Type == ItemTypeTest {
				break // Found a selectable test item
			}
			if c.selectedIndex < len(c.displayItems)-1 {
				c.selectedIndex++
			} else {
				// Can't go further down, revert
				c.selectedIndex = originalIndex
				return
			}
		}

		// Update view and rebuild
		if c.selectedIndex >= c.visibleStart+c.listHeight {
			c.visibleStart = c.selectedIndex - c.listHeight + 1
		}
		if c.selectedIndex != c.lastSelectedIndex {
			c.lastSelectedIndex = c.selectedIndex
		}
		c.buildItems()
	}
}
