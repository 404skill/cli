package login

import (
	"context"
	"strings"

	"404skill-cli/auth"
	"404skill-cli/tui/components/footer"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Component handles user authentication UI
type Component struct {
	inputs      []textinput.Model
	focusIdx    int
	errorMsg    string
	loggingIn   bool
	authService *auth.AuthService
	footer      *footer.Component
}

// New creates a new login component with dependency injection
func New(authProvider auth.AuthProvider, configWriter auth.ConfigWriter) *Component {
	username := textinput.New()
	username.Placeholder = "Username"
	username.Focus()
	username.CharLimit = 64
	username.Width = 32

	password := textinput.New()
	password.Placeholder = "Password"
	password.EchoMode = textinput.EchoPassword
	password.EchoCharacter = '•'
	password.CharLimit = 64
	password.Width = 32

	return &Component{
		inputs:      []textinput.Model{username, password},
		focusIdx:    0,
		authService: auth.NewAuthService(authProvider, configWriter),
		footer:      footer.New(),
	}
}

// Init initializes the login component
func (c *Component) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the login component
func (c *Component) Update(msg tea.Msg) (*Component, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "shift+tab":
			if msg.String() == "shift+tab" {
				c.focusIdx--
			} else {
				c.focusIdx++
			}
			if c.focusIdx > 1 {
				c.focusIdx = 0
			} else if c.focusIdx < 0 {
				c.focusIdx = 1
			}
			c.updateFocus()
			return c, nil
		case "enter":
			if c.focusIdx == 1 && !c.loggingIn {
				c.loggingIn = true
				c.errorMsg = ""
				return c, c.tryLogin()
			}
			c.focusIdx = 1
			c.updateFocus()
			return c, nil
		default:
			// Pass all other keys to the focused input only
			c.inputs[c.focusIdx], cmd = c.inputs[c.focusIdx].Update(msg)
			return c, cmd
		}
	case LoginSuccessMsg:
		c.errorMsg = ""
		c.loggingIn = false
		return c, LoginSuccessCommand()
	case LoginErrorMsg:
		c.errorMsg = msg.Error
		c.loggingIn = false
		return c, nil
	}

	return c, nil
}

// GetUsername returns the current username input
func (c *Component) GetUsername() string {
	return c.inputs[0].Value()
}

// GetPassword returns the current password input
func (c *Component) GetPassword() string {
	return c.inputs[1].Value()
}

// SetError sets the error message
func (c *Component) SetError(msg string) {
	c.errorMsg = msg
}

// SetLoggingIn sets the logging in state
func (c *Component) SetLoggingIn(state bool) {
	c.loggingIn = state
}

// View renders the login component
func (c *Component) View() string {
	var inputs []string
	for i := range c.inputs {
		input := c.inputs[i].View()
		if i == c.focusIdx {
			accent := lipgloss.Color("#00ffaa")
			input += lipgloss.NewStyle().Foreground(accent).Render("█")
		}
		inputs = append(inputs, input)
	}

	loginBoxStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ff00")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00ffaa")).
		Padding(1, 4).
		Width(44)

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ff0000")).
		Bold(true)

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ffaa")).
		Bold(true)

	content := "Username: " + inputs[0] + "\n" +
		"Password: " + inputs[1] + "\n" +
		strings.Repeat(" ", 2) + c.footer.View(footer.TabBinding, footer.SubmitBinding, footer.QuitBinding)

	if c.errorMsg != "" {
		content += "\n" + errorStyle.Render(c.errorMsg)
	}
	if c.loggingIn {
		content += "\n" + headerStyle.Render("Logging in...")
	}

	loginBox := loginBoxStyle.Render(content)

	// Add ASCII art header
	asciiArt := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ff00")).Render(`
/==============================================================================================\
||                                                                                            ||
||      ___   ___  ________  ___   ___  ________  ___  __    ___  ___       ___               ||
||     |\  \ |\  \|\   __  \|\  \ |\  \|\   ____\|\  \|\  \ |\  \|\  \     |\  \              ||
||     \ \  \\_\  \ \  \|\  \ \  \\_\  \ \  \___|\ \  \/  /|\ \  \ \  \    \ \  \             ||
||      \ \______  \ \  \\\  \ \______  \ \_____  \ \   ___  \ \  \ \  \    \ \  \            ||
||       \|_____|\  \ \  \\\  \|_____|\  \|____|\  \ \  \\ \  \ \  \ \  \____\ \  \____       ||
||              \ \__\ \_______\     \ \__\____\_\  \ \__\\ \__\ \__\ \_______\ \_______\     ||
||               \|__|\|_______|      \|__|\_________\|__| \|__|\|__|\|_______|\|_______|     ||
||                                        \|_________|                                        ||
||                                                                                            ||
\==============================================================================================/
                                                                       `)

	// Center the login box on screen
	termWidth, termHeight := 80, 24
	boxLines := strings.Split(loginBox, "\n")
	boxHeight := len(boxLines)
	padTop := (termHeight - boxHeight) / 2
	padLeft := (termWidth - 44) / 2 // 44 is the box width

	centered := strings.Repeat("\n", padTop) +
		asciiArt + "\n\n" +
		strings.Repeat(" ", padLeft) + strings.Join(boxLines, "\n"+strings.Repeat(" ", padLeft))

	return centered
}

// updateFocus updates which input has focus
func (c *Component) updateFocus() {
	for i := 0; i < len(c.inputs); i++ {
		if i == c.focusIdx {
			c.inputs[i].Focus()
		} else {
			c.inputs[i].Blur()
		}
	}
}

// tryLogin attempts to log in with the current credentials
// Uses the AuthService for business logic
func (c *Component) tryLogin() tea.Cmd {
	return func() tea.Msg {
		username := c.inputs[0].Value()
		password := c.inputs[1].Value()

		// Use the auth service for business logic
		result := c.authService.AttemptLogin(context.Background(), username, password)

		if result.Success {
			return LoginSuccessMsg{}
		} else {
			return LoginErrorMsg{Error: result.Error}
		}
	}
}
