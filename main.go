package main

import (
	"404skill-cli/api"
	"404skill-cli/config"
	"404skill-cli/tui"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Create API client with config token provider
	tokenProvider := config.NewConfigTokenProvider()
	client, err := api.NewClient(tokenProvider)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating API client: %v\n", err)
		os.Exit(1)
	}

	// Initialize and run the TUI
	p := tea.NewProgram(tui.InitialModel(client), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		os.Exit(1)
	}
}
