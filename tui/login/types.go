package login

import (
	tea "github.com/charmbracelet/bubbletea"
)

// LoginSuccessMsg is sent when login is successful
type LoginSuccessMsg struct{}

// LoginErrorMsg is sent when login fails
type LoginErrorMsg struct {
	Error string
}

// LoginSuccessCommand creates a command that signals successful login
func LoginSuccessCommand() tea.Cmd {
	return func() tea.Msg {
		return LoginSuccessMsg{}
	}
}

// SessionExpiredMsg is sent when session needs refresh
type SessionExpiredMsg struct {
	Error string
}
