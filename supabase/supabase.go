package supabase

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/supabase-community/supabase-go"
)

// These two vars are empty by default. We will override them via -ldflags in production builds.
var (
	embeddedSupabaseURL string
	embeddedSupabaseKey string
)

func NewSupabaseClient() (*supabase.Client, error) {
	if embeddedSupabaseURL != "" && embeddedSupabaseKey != "" {
		return supabase.NewClient(embeddedSupabaseURL, embeddedSupabaseKey, nil)
	}

	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("failed to load environment: %w", err)
	}

	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseKey := os.Getenv("SUPABASE_KEY")

	if supabaseURL == "" || supabaseKey == "" {
		return nil, fmt.Errorf("SUPABASE_URL and SUPABASE_KEY must be set in environment variables")
	}

	return supabase.NewClient(supabaseURL, supabaseKey, nil)
}
