package menu

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Component represents a vertical menu with selectable items
type Component struct {
	items         []string
	selectedIndex int
	styles        Styles
}

// Styles defines the visual styling for menu components
type Styles struct {
	ItemStyle      lipgloss.Style
	SelectedStyle  lipgloss.Style
	Cursor         string
	SelectedCursor string
}

// DefaultStyles returns the default styling for menus that matches the application theme
func DefaultStyles() Styles {
	// Colors from the main application theme
	primary := lipgloss.Color("#00ff00") // Bright green
	bg := lipgloss.Color("#000000")      // Black

	return Styles{
		ItemStyle: lipgloss.NewStyle().
			Foreground(primary).
			Background(bg).
			Padding(0, 1),
		SelectedStyle: lipgloss.NewStyle().
			Foreground(bg).
			Background(primary).
			Bold(true).
			Padding(0, 1),
		Cursor:         "  ",
		SelectedCursor: "> ",
	}
}

// New creates a new menu component with the given items
func New(items []string) *Component {
	return &Component{
		items:         items,
		selectedIndex: 0,
		styles:        DefaultStyles(),
	}
}

// SetItems updates the menu items
func (c *Component) SetItems(items []string) {
	c.items = items
	// Reset selection if it's out of bounds
	if c.selectedIndex >= len(items) {
		c.selectedIndex = 0
	}
}

// GetItems returns the current menu items
func (c *Component) GetItems() []string {
	return c.items
}

// SetSelectedIndex sets the current selection
func (c *Component) SetSelectedIndex(index int) {
	if index >= 0 && index < len(c.items) {
		c.selectedIndex = index
	}
}

// GetSelectedIndex returns the current selection index
func (c *Component) GetSelectedIndex() int {
	return c.selectedIndex
}

// GetSelectedItem returns the currently selected item
func (c *Component) GetSelectedItem() string {
	if len(c.items) == 0 || c.selectedIndex < 0 || c.selectedIndex >= len(c.items) {
		return ""
	}
	return c.items[c.selectedIndex]
}

// SetStyles updates the menu styling
func (c *Component) SetStyles(styles Styles) {
	c.styles = styles
}

// MenuSelectMsg is sent when an item is selected (Enter pressed)
type MenuSelectMsg struct {
	SelectedIndex int
	SelectedItem  string
}

// Update handles keyboard input for menu navigation
func (c *Component) Update(msg tea.Msg) (*Component, tea.Cmd) {
	if len(c.items) == 0 {
		return c, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			c.selectedIndex--
			if c.selectedIndex < 0 {
				c.selectedIndex = len(c.items) - 1
			}
		case "down", "j":
			c.selectedIndex++
			if c.selectedIndex >= len(c.items) {
				c.selectedIndex = 0
			}
		case "enter":
			return c, func() tea.Msg {
				return MenuSelectMsg{
					SelectedIndex: c.selectedIndex,
					SelectedItem:  c.GetSelectedItem(),
				}
			}
		}
	}

	return c, nil
}

// View renders the menu
func (c *Component) View() string {
	if len(c.items) == 0 {
		return ""
	}

	var menu string
	for i, item := range c.items {
		cursor := c.styles.Cursor
		style := c.styles.ItemStyle

		if i == c.selectedIndex {
			cursor = c.styles.SelectedCursor
			style = c.styles.SelectedStyle
		}

		menu += fmt.Sprintf("%s%s\n", cursor, style.Render(item))
	}

	// Remove trailing newline
	if len(menu) > 0 {
		menu = menu[:len(menu)-1]
	}

	return menu
}

// IsEmpty returns true if the menu has no items
func (c *Component) IsEmpty() bool {
	return len(c.items) == 0
}
