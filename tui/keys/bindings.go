package keys

import (
	"404skill-cli/tui/components/footer"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// GlobalKeyMap defines global key bindings used across the application
type GlobalKeyMap struct {
	Up    key.Binding
	Down  key.Binding
	Enter key.Binding
	Quit  key.Binding
	Back  key.Binding
	Tab   key.Binding
}

// DefaultGlobalKeys returns the default global key bindings
func DefaultGlobalKeys() GlobalKeyMap {
	return GlobalKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "b"),
			key.WithHelp("esc/b", "back"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab", "shift+tab"),
			key.WithHelp("tab", "switch"),
		),
	}
}

// Handler provides a centralized way to handle common key patterns
type Handler struct {
	keys GlobalKeyMap
}

// NewHandler creates a new key handler with default bindings
func NewHandler() *Handler {
	return &Handler{
		keys: DefaultGlobalKeys(),
	}
}

// HandleGlobalKeys handles global keys that should work in any state
func (h *Handler) HandleGlobalKeys(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, h.keys.Quit):
		return tea.Quit
	}
	return nil
}

// IsQuit returns true if the key message is a quit command
func (h *Handler) IsQuit(msg tea.KeyMsg) bool {
	return key.Matches(msg, h.keys.Quit)
}

// IsBack returns true if the key message is a back command
func (h *Handler) IsBack(msg tea.KeyMsg) bool {
	return key.Matches(msg, h.keys.Back)
}

// IsEnter returns true if the key message is an enter command
func (h *Handler) IsEnter(msg tea.KeyMsg) bool {
	return key.Matches(msg, h.keys.Enter)
}

// IsUp returns true if the key message is an up command
func (h *Handler) IsUp(msg tea.KeyMsg) bool {
	return key.Matches(msg, h.keys.Up)
}

// IsDown returns true if the key message is a down command
func (h *Handler) IsDown(msg tea.KeyMsg) bool {
	return key.Matches(msg, h.keys.Down)
}

// IsTab returns true if the key message is a tab command
func (h *Handler) IsTab(msg tea.KeyMsg) bool {
	return key.Matches(msg, h.keys.Tab)
}

// FooterBindings returns appropriate footer bindings for different contexts
type FooterBindings struct{}

// NewFooterBindings creates a new footer bindings helper
func NewFooterBindings() *FooterBindings {
	return &FooterBindings{}
}

// Navigation returns bindings for navigation contexts
func (f *FooterBindings) Navigation() []footer.KeyBinding {
	return []footer.KeyBinding{
		footer.NavigateBinding,
		footer.EnterBinding,
		footer.QuitBinding,
	}
}

// NavigationWithBack returns bindings for navigation contexts with back option
func (f *FooterBindings) NavigationWithBack() []footer.KeyBinding {
	return []footer.KeyBinding{
		footer.NavigateBinding,
		footer.EnterBinding,
		footer.BackBinding,
		footer.QuitBinding,
	}
}

// Login returns bindings for login context
func (f *FooterBindings) Login() []footer.KeyBinding {
	return []footer.KeyBinding{
		footer.TabBinding,
		footer.SubmitBinding,
		footer.QuitBinding,
	}
}

// Download returns bindings for download context
func (f *FooterBindings) Download() []footer.KeyBinding {
	return []footer.KeyBinding{
		footer.BackBinding,
		footer.QuitBinding,
	}
}
