package commands

import (
	"404skill-cli/api"
	"404skill-cli/template"
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// InitCmd represents the init command
type InitCmd struct {
	client     api.ClientInterface
	downloader template.DownloaderInterface
	ProjectID  string `short:"p" long:"project" description:"Project ID or name to initialize"`
}

// NewInitCmd creates a new instance of InitCmd
func NewInitCmd(client api.ClientInterface) *InitCmd {
	return &InitCmd{
		client:     client,
		downloader: template.NewDownloader(),
	}
}

// Execute runs the init command
func (c *InitCmd) Execute(args []string) error {
	if c.ProjectID == "" {
		return fmt.Errorf("project ID or name is required")
	}

	// Get project template from API
	template, err := c.client.InitProject(context.Background(), c.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to initialize project: %w", err)
	}

	// Create project directory
	projectDir := filepath.Join(".", template.ProjectName)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Download and extract template
	extractedDir, err := c.downloader.DownloadAndExtract(template.DownloadURL, projectDir)
	if err != nil {
		return fmt.Errorf("failed to download and extract template: %w", err)
	}

	fmt.Printf("Project initialized successfully at: %s\n", extractedDir)
	return nil
}
