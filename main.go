package main

import (
	"404skill-cli/api"
	"404skill-cli/auth"
	"404skill-cli/config"
	"404skill-cli/supabase"
	"404skill-cli/tracing"
	"404skill-cli/tui"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// USE THIS TO SHOW VERSION INFO FOR THE USER (https://goreleaser.com/cookbooks/using-main.version/?h=ldflag)
var (
	version = "dev"
)

func main() {
	// Initialize tracing system
	tracingConfig := tracing.DefaultConfig()
	tracingConfig.LocalDir = "~/.404skill/traces"
	// Use a shorter timeout and disable uploads for faster quit in development
	tracingConfig.UploadTimeout = 2 * time.Second
	tracingConfig.UploadEndpoint = "" // Disable uploads for development

	if err := tracing.InitGlobalTracingWithVersion(tracingConfig, version); err != nil {
		// Don't fail the application if tracing fails to initialize
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize tracing: %v\n", err)
	}

	// Ensure tracing is properly closed on exit
	defer func() {
		if err := tracing.CloseGlobalTracing(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to close tracing: %v\n", err)
		}
	}()

	// Track application startup
	startupTracker := tracing.TimedOperation("application_startup")
	startupTracker.AddMetadata("version", version)

	// Create auth dependencies
	supabaseClient, err := supabase.NewSupabaseClient()
	if err != nil {
		_ = tracing.TrackError(err, "main")
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
		_ = tracing.TrackError(err, "main")
		fmt.Fprintf(os.Stderr, "Error creating API client: %v\n", err)
		os.Exit(1)
	}

	// Initialize the TUI model
	model, err := tui.InitialModel(client, version)
	if err != nil {
		_ = tracing.TrackError(err, "main")
		fmt.Fprintf(os.Stderr, "Error initializing TUI: %v\n", err)
		os.Exit(1)
	}

	// Complete startup tracking
	_ = startupTracker.Complete()

	// Track TUI launch
	_ = tracing.TrackStateTransition("application_startup", "tui_launched", "initialization_complete")

	// Run the TUI
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		_ = tracing.TrackError(err, "main")
		os.Exit(1)
	}

	// Track application exit
	_ = tracing.TrackStateTransition("tui_active", "application_exit", "normal_shutdown")
}
