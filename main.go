package main

import (
	"404skill-cli/api"
	"404skill-cli/auth"
	"404skill-cli/config"
	"404skill-cli/supabase"
	"404skill-cli/tui"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// USE THIS TO SHOW VERSION INFO FOR THE USER (https://goreleaser.com/cookbooks/using-main.version/?h=ldflag)
var (
	version string
)

func main() {
	// Create auth dependencies
	supabaseClient, err := supabase.NewSupabaseClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Supabase client: %v\n", err)
		os.Exit(1)
	}

	authProvider := auth.NewSupabaseAuth(supabaseClient)
	configWriter := config.SimpleConfigWriter{}
	authService := auth.NewAuthService(authProvider, &configWriter)

	// Create API client with config manager as token provider
	configManager := config.NewConfigManager(authService)
	client, err := api.NewClient(configManager)
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
