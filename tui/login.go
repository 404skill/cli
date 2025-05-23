package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"404skill-cli/auth"
	"404skill-cli/config"
	"404skill-cli/supabase"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LoginComponent handles the login state and operations
type LoginComponent struct {
	inputs     []textinput.Model
	focusIdx   int
	errorMsg   string
	loggingIn  bool
	authClient auth.AuthProvider
}

// NewLoginComponent creates a new login component
func NewLoginComponent() *LoginComponent {
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

	client, _ := supabase.NewSupabaseClient()
	return &LoginComponent{
		inputs:     []textinput.Model{username, password},
		focusIdx:   0,
		authClient: auth.NewSupabaseAuth(client),
	}
}

// Update handles messages for the login component
func (l *LoginComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "shift+tab":
			if msg.String() == "shift+tab" {
				l.focusIdx--
			} else {
				l.focusIdx++
			}
			if l.focusIdx > 1 {
				l.focusIdx = 0
			} else if l.focusIdx < 0 {
				l.focusIdx = 1
			}
			for i := 0; i < len(l.inputs); i++ {
				if i == l.focusIdx {
					l.inputs[i].Focus()
				} else {
					l.inputs[i].Blur()
				}
			}
			return l, nil
		case "enter":
			if l.focusIdx == 1 {
				l.loggingIn = true
				l.errorMsg = ""
				return l, l.tryLogin()
			}
			l.focusIdx = 1
			for i := 0; i < len(l.inputs); i++ {
				if i == l.focusIdx {
					l.inputs[i].Focus()
				} else {
					l.inputs[i].Blur()
				}
			}
			return l, nil
		default:
			l.inputs[l.focusIdx], cmd = l.inputs[l.focusIdx].Update(msg)
			return l, cmd
		}
	case string:
		if msg == "login-success" {
			l.errorMsg = ""
			l.loggingIn = false
		}
	case errMsg:
		l.errorMsg = msg.err.Error()
		l.loggingIn = false
	}

	return l, nil
}

// View renders the login component
func (l *LoginComponent) View() string {
	inputs := []string{}
	for i := range l.inputs {
		input := l.inputs[i].View()
		if i == l.focusIdx {
			input += lipgloss.NewStyle().Foreground(accent).Render("█")
		}
		inputs = append(inputs, input)
	}

	loginBox := loginBoxStyle.Render(
		"Username: " + inputs[0] + "\n" +
			"Password: " + inputs[1] + "\n" +
			strings.Repeat(" ", 2) + "[Tab] Switch  [Enter] Submit  [q] Quit" +
			func() string {
				if l.errorMsg != "" {
					return "\n" + errorStyle.Render(l.errorMsg)
				}
				if l.loggingIn {
					return "\n" + headerStyle.Render("Logging in...")
				}
				return ""
			}(),
	)

	return loginBox
}

// tryLogin attempts to log in with the current credentials
func (l *LoginComponent) tryLogin() tea.Cmd {
	return func() tea.Msg {
		username := l.inputs[0].Value()
		password := l.inputs[1].Value()

		token, err := l.authClient.SignIn(context.Background(), username, password)
		if err != nil {
			return errMsg{err: fmt.Errorf("invalid credentials: %w", err)}
		}

		cfg := config.Config{
			Username:    username,
			Password:    password,
			AccessToken: token,
			LastUpdated: time.Now(),
		}
		_ = config.WriteConfig(cfg)
		return "login-success"
	}
}
