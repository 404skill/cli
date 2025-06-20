package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"404skill-cli/config"
)

// TokenProvider defines the interface for token management
type TokenProvider interface {
	GetToken() (string, error)
}

// ClientInterface defines the interface for API client operations
type ClientInterface interface {
	ListProjects(ctx context.Context) ([]Project, error)
	BulkUpdateProfileTests(ctx context.Context, failed, passed []string, projectID string) error
	InitializeProject(ctx context.Context, projectId string) error
}

// Client represents the API client
type Client struct {
	httpClient    *http.Client
	baseURL       string
	tokenProvider TokenProvider
}

// Project represents a project in the system
type Project struct {
	ID                         string `json:"id"`
	Name                       string `json:"name"`
	Difficulty                 string `json:"difficulty"`
	Language                   string `json:"language"`
	Description                string `json:"description"`
	RepoUrl                    string `json:"repo_url"`
	Type                       string `json:"type"`
	EstimatedDurationInMinutes int    `json:"estimated_duration_minutes"`
	AccessTier                 string `json:"access_tier"`
	Technologies               string `json:"technologies"`
}

// ProjectTemplate represents a project template response
type ProjectTemplate struct {
	DownloadURL string `json:"download_url"`
	ProjectName string `json:"project_name"`
}

// NewClient creates a new API client
func NewClient(tokenProvider TokenProvider) (*Client, error) {
	// Call GetBaseURL() and catch its error
	baseURL, err := config.GetBaseURL()
	if err != nil {
		return nil, err
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL:       baseURL,
		tokenProvider: tokenProvider,
	}, nil
}

// ListProjects retrieves all projects
func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
	token, err := c.tokenProvider.GetToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/projects", c.baseURL), nil)
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

// Registers user has started a project
func (c *Client) InitializeProject(ctx context.Context, projectId string) error {
	token, err := c.tokenProvider.GetToken()
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/profile-projects?projectId=%s", c.baseURL, projectId), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

type BulkUpdateRequest struct {
	ProjectId       string   `json:"projectId"`
	FailedTestNames []string `json:"failedTestNames"`
	PassedTestNames []string `json:"passedTestNames"`
}

func (c *Client) BulkUpdateProfileTests(ctx context.Context, failed, passed []string, projectID string) error {
	token, err := c.tokenProvider.GetToken()
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	reqBody := BulkUpdateRequest{
		FailedTestNames: failed,
		PassedTestNames: passed,
		ProjectId:       projectID,
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/profile-tests/bulk-update", c.baseURL), bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s, %s", resp.Status, string(bodyBytes))
	}
	return nil
}
