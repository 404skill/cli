package controller

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Command message types
type (
	// TokenRefreshMsg is sent when token refresh completes
	TokenRefreshMsg struct {
		Error error
	}

	// VersionCheckMsg is sent when version check completes
	VersionCheckMsg struct {
		Info VersionInfo
	}

	// VersionTickerMsg is sent periodically to check for updates
	VersionTickerMsg struct{}
)

// refreshTokenCmd attempts to refresh the authentication token
func (c *Controller) refreshTokenCmd() tea.Cmd {
	return func() tea.Msg {
		// Use the config manager's GetToken method which handles refresh automatically
		_, err := c.configManager.GetToken()
		return TokenRefreshMsg{Error: err}
	}
}

// checkVersionCmd checks for version updates
func (c *Controller) checkVersionCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		info := c.versionChecker.CheckForUpdates(ctx)
		return VersionCheckMsg{Info: info}
	}
}

// versionTickerCmd creates a periodic version check
func (c *Controller) versionTickerCmd() tea.Cmd {
	return tea.Tick(30*time.Minute, func(t time.Time) tea.Msg {
		return VersionTickerMsg{}
	})
}
