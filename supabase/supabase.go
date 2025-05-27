package supabase

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/supabase-community/supabase-go"
)

func NewSupabaseClient() (*supabase.Client, error) {
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
