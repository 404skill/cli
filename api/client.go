package api

import (
    "context"
    "fmt"
    "net/http"
    "time"
    "encoding/json"
)

// TokenProvider defines the interface for token management
type TokenProvider interface {
    GetToken() (string, error)
}

// ClientInterface defines the interface for API client operations
type ClientInterface interface {
    ListProjects(ctx context.Context) ([]Project, error)
}

// Client represents the API client
type Client struct {
    httpClient    *http.Client
    baseURL       string
    tokenProvider TokenProvider
}

// NewClient creates a new API client
func NewClient(tokenProvider TokenProvider) *Client {
    return &Client{
        httpClient: &http.Client{
            Timeout: 10 * time.Second,
        },
        baseURL:       "http://localhost:3000",
        tokenProvider: tokenProvider,
    }
}

// Project represents a project in the system
type Project struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

// ListProjects retrieves all projects
func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
    token, err := c.tokenProvider.GetToken()
    if err != nil {
        return nil, fmt.Errorf("failed to get token: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/hello", c.baseURL), nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to make request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
    }

    var projects []Project
    if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }

    return projects, nil
} 