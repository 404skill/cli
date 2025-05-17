package supabase

import (
    "fmt"
    "os"
    "github.com/supabase-community/supabase-go"
)

func NewSupabaseClient() (*supabase.Client, error) {
    supabaseUrl := os.Getenv("SUPABASE_URL")
    supabaseKey := os.Getenv("SUPABASE_KEY")
    if supabaseUrl == "" || supabaseKey == "" {
        return nil, fmt.Errorf("Supabase credentials are not set")
    }

    client, err := supabase.NewClient(supabaseUrl, supabaseKey, nil)
    if err != nil {
        return nil, fmt.Errorf("cannot initialize client: %w", err)
    }

    return client, nil
} 