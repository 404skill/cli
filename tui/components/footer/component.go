package footer

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Component represents a footer with help text
type Component struct {
	style lipgloss.Style
}

// New creates a new footer component
func New() *Component {
	return &Component{
		style: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00aa00")). // Darker green (secondary)
			Faint(true),
	}
}

// KeyBinding represents a single key binding
type KeyBinding struct {
	Key         string
	Description string
}

// View renders the footer with the provided key bindings
func (c *Component) View(bindings ...KeyBinding) string {
	if len(bindings) == 0 {
		return ""
	}

	var parts []string
	for _, binding := range bindings {
		parts = append(parts, binding.Format())
	}

	return c.style.Render(strings.Join(parts, "  "))
}

// Format renders a key binding in the standard format
func (kb KeyBinding) Format() string {
	if kb.Key == "" || kb.Description == "" {
		return ""
	}
	return "[" + kb.Key + "] " + kb.Description
}

// Common key bindings for reuse
var (
	QuitBinding     = KeyBinding{Key: "q", Description: "quit"}
	BackBinding     = KeyBinding{Key: "esc/b", Description: "back"}
	EnterBinding    = KeyBinding{Key: "enter", Description: "select"}
	ConfirmBinding  = KeyBinding{Key: "enter", Description: "confirm"}
	SubmitBinding   = KeyBinding{Key: "enter", Description: "submit"}
	TabBinding      = KeyBinding{Key: "tab", Description: "switch"}
	NavigateBinding = KeyBinding{Key: "↑/↓ or k/j", Description: "move"}
)
