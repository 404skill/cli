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
	repoURL := fmt.Sprintf("https://github.com/404skill/%s_%s", repoName, project.ID)
	targetDir := filepath.Join(projectsDir, fmt.Sprintf("%s_%s", repoName, project.ID))

	// Create progress callback for main project (0-50%)
	mainProgressCallback := func(progress float64) {
		if progressCallback != nil {
			progressCallback(progress * 0.5) // Scale to 0-50%
		}
	}

	// Clone main project repository
	if err := g.cloneMainProject(ctx, repoURL, targetDir, mainProgressCallback); err != nil {
		return err
	}

	// Create progress callback for test project (50-100%)
	testProgressCallback := func(progress float64) {
		if progressCallback != nil {
			progressCallback(0.5 + (progress * 0.5)) // Scale to 50-100%
		}
	}

	if err := g.cloneTestProject(ctx, repoName, project.ID, projectsDir, testProgressCallback); err != nil {
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
	var lastProgress float64 = 0

	for scanner.Scan() {
		line := scanner.Text()

		// Parse different types of progress output
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
						lastProgress = progress / 100
						if progressCallback != nil {
							progressCallback(lastProgress)
						}
					}
				}
			}
		} else if strings.Contains(line, "Resolving deltas") {
			// Parse delta resolution progress
			if strings.Contains(line, "%") {
				parts := strings.Split(line, "%")
				if len(parts) > 0 {
					progressStr := strings.TrimSpace(parts[0])
					if spaceIdx := strings.LastIndex(progressStr, " "); spaceIdx != -1 {
						progressStr = progressStr[spaceIdx+1:]
					}
					if progress, err := strconv.ParseFloat(progressStr, 64); err == nil {
						// Delta resolution is typically the last 20% of the process
						deltaProgress := (progress / 100) * 0.2
						lastProgress = 0.8 + deltaProgress
						if progressCallback != nil {
							progressCallback(lastProgress)
						}
					}
				}
			}
		} else if strings.Contains(line, "Cloning into") {
			// Initial cloning message
			if progressCallback != nil {
				progressCallback(0.0)
			}
		} else if strings.Contains(line, "remote: Counting objects") {
			// Counting objects phase
			if progressCallback != nil {
				progressCallback(0.1)
			}
		} else if strings.Contains(line, "remote: Compressing objects") {
			// Compressing objects phase
			if progressCallback != nil {
				progressCallback(0.2)
			}
		} else if strings.Contains(line, "Unpacking objects") {
			// Unpacking objects phase
			if progressCallback != nil {
				progressCallback(0.6)
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

	// Ensure we reach 100% when complete
	if progressCallback != nil {
		progressCallback(1.0)
	}

	return nil
}

// cloneTestProject clones the test repository
func (g *GitDownloader) cloneTestProject(ctx context.Context, repoName, projectID, projectsDir string, progressCallback ProgressCallback) error {
	testRepoURL := fmt.Sprintf("https://github.com/404skill/%s_test_%s", repoName, projectID)
	testDir := filepath.Join(projectsDir, ".tests", fmt.Sprintf("%s_%s", repoName, projectID))

	// Create tests directory
	if err := g.fileManager.CreateDirectory(filepath.Dir(testDir)); err != nil {
		return fmt.Errorf("failed to create tests directory: %w", err)
	}

	// Remove existing test directory if it exists
	if err := g.fileManager.RemoveDirectory(testDir); err != nil {
		return fmt.Errorf("failed to remove existing test directory: %w", err)
	}

	// Start git clone with progress output
	cmd := exec.CommandContext(ctx, "git", "clone", "--progress", testRepoURL, testDir)
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
	var lastProgress float64 = 0

	for scanner.Scan() {
		line := scanner.Text()

		// Parse different types of progress output
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
						lastProgress = progress / 100
						if progressCallback != nil {
							progressCallback(lastProgress)
						}
					}
				}
			}
		} else if strings.Contains(line, "Resolving deltas") {
			// Parse delta resolution progress
			if strings.Contains(line, "%") {
				parts := strings.Split(line, "%")
				if len(parts) > 0 {
					progressStr := strings.TrimSpace(parts[0])
					if spaceIdx := strings.LastIndex(progressStr, " "); spaceIdx != -1 {
						progressStr = progressStr[spaceIdx+1:]
					}
					if progress, err := strconv.ParseFloat(progressStr, 64); err == nil {
						// Delta resolution is typically the last 20% of the process
						deltaProgress := (progress / 100) * 0.2
						lastProgress = 0.8 + deltaProgress
						if progressCallback != nil {
							progressCallback(lastProgress)
						}
					}
				}
			}
		} else if strings.Contains(line, "Cloning into") {
			// Initial cloning message
			if progressCallback != nil {
				progressCallback(0.0)
			}
		} else if strings.Contains(line, "remote: Counting objects") {
			// Counting objects phase
			if progressCallback != nil {
				progressCallback(0.1)
			}
		} else if strings.Contains(line, "remote: Compressing objects") {
			// Compressing objects phase
			if progressCallback != nil {
				progressCallback(0.2)
			}
		} else if strings.Contains(line, "Unpacking objects") {
			// Unpacking objects phase
			if progressCallback != nil {
				progressCallback(0.6)
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

	// Ensure we reach 100% when complete
	if progressCallback != nil {
		progressCallback(1.0)
	}

	return nil
}
