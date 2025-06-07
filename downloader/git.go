package downloader

import (
	"404skill-cli/api"
	"404skill-cli/config"
	"404skill-cli/filesystem"
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// GitDownloader implements Downloader using git clone
type GitDownloader struct {
	fileManager   *filesystem.Manager
	configManager *config.ConfigManager
	apiClient     api.ClientInterface
}

// NewGitDownloader creates a new Git-based downloader
func NewGitDownloader(fileManager *filesystem.Manager, configManager *config.ConfigManager, apiClient api.ClientInterface) *GitDownloader {
	return &GitDownloader{
		fileManager:   fileManager,
		configManager: configManager,
		apiClient:     apiClient,
	}
}

// DownloadProject downloads a project using git clone
func (g *GitDownloader) DownloadProject(ctx context.Context, project *api.Project, language string, progressCallback ProgressCallback) error {
	// Create projects directory if it doesn't exist
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	projectsDir := filepath.Join(homeDir, "404skill_projects")
	if err := g.fileManager.CreateDirectory(projectsDir); err != nil {
		return fmt.Errorf("failed to create projects directory: %w", err)
	}

	// Format project name for repo URL
	repoName := strings.ToLower(strings.ReplaceAll(project.Name, " ", "_"))
	repoURL := fmt.Sprintf("https://github.com/404skill/%s_%s", repoName, language)
	targetDir := filepath.Join(projectsDir, fmt.Sprintf("%s_%s", repoName, language))

	// Clone both main project and test repository
	if err := g.cloneMainProject(ctx, repoURL, targetDir, progressCallback); err != nil {
		return err
	}

	if err := g.cloneTestProject(ctx, repoName, language, projectsDir); err != nil {
		return err
	}

	// Verify the clone was successful
	if !g.fileManager.DirectoryExists(targetDir) {
		return fmt.Errorf("clone appeared to succeed but target directory is missing")
	}

	// Update config with downloaded project
	if err := g.configManager.UpdateDownloadedProject(project.ID); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

	// Initialize project in API
	if err := g.apiClient.InitializeProject(ctx, project.ID); err != nil {
		return fmt.Errorf("failed to initialize project: %w", err)
	}

	// Open file explorer at the cloned directory
	if err := g.fileManager.OpenFileExplorer(targetDir); err != nil {
		// Don't return error here, as the download was successful
		fmt.Printf("Warning: Failed to open file explorer: %v\n", err)
	}

	return nil
}

// cloneMainProject clones the main project repository
func (g *GitDownloader) cloneMainProject(ctx context.Context, repoURL, targetDir string, progressCallback ProgressCallback) error {
	// Remove existing directory if it exists
	if err := g.fileManager.RemoveDirectory(targetDir); err != nil {
		return fmt.Errorf("failed to remove existing directory: %w", err)
	}

	// Start git clone with progress output
	cmd := exec.CommandContext(ctx, "git", "clone", "--progress", repoURL, targetDir)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start git clone: %w", err)
	}

	// Read progress from stderr
	scanner := bufio.NewScanner(stderr)
	var cloneError string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Receiving objects") {
			// Parse percentage from line like "Receiving objects: 45% (9/20)"
			if strings.Contains(line, "%") {
				parts := strings.Split(line, "%")
				if len(parts) > 0 {
					// Extract just the number part
					progressStr := strings.TrimSpace(parts[0])
					// Find the last space and take everything after it
					if spaceIdx := strings.LastIndex(progressStr, " "); spaceIdx != -1 {
						progressStr = progressStr[spaceIdx+1:]
					}
					if progress, err := strconv.ParseFloat(progressStr, 64); err == nil {
						if progressCallback != nil {
							progressCallback(progress / 100)
						}
					}
				}
			}
		} else if strings.Contains(line, "error:") || strings.Contains(line, "fatal:") {
			cloneError = line
		}
	}

	if err := cmd.Wait(); err != nil {
		if cloneError != "" {
			return fmt.Errorf("git clone failed: %s", cloneError)
		}
		return fmt.Errorf("git clone failed: %w", err)
	}

	return nil
}

// cloneTestProject clones the test repository
func (g *GitDownloader) cloneTestProject(ctx context.Context, repoName, language, projectsDir string) error {
	testRepoURL := fmt.Sprintf("https://github.com/404skill/%s_%s_test", repoName, language)
	testDir := filepath.Join(projectsDir, ".tests", fmt.Sprintf("%s_%s", repoName, language))

	// Create tests directory
	if err := g.fileManager.CreateDirectory(filepath.Dir(testDir)); err != nil {
		return fmt.Errorf("failed to create tests directory: %w", err)
	}

	// Remove existing test directory if it exists
	if err := g.fileManager.RemoveDirectory(testDir); err != nil {
		return fmt.Errorf("failed to remove existing test directory: %w", err)
	}

	// Clone test repository
	cmd := exec.CommandContext(ctx, "git", "clone", "--progress", testRepoURL, testDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone test repository: %w", err)
	}

	return nil
}
