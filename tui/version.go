package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// VersionInfo represents version information
type VersionInfo struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	Error           error
}

// VersionChecker handles version checking operations
type VersionChecker struct {
	httpClient *http.Client
}

// NPMResponse represents the npm registry response
type NPMResponse struct {
	DistTags struct {
		Latest string `json:"latest"`
	} `json:"dist-tags"`
}

// NewVersionChecker creates a new version checker
func NewVersionChecker() *VersionChecker {
	return &VersionChecker{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CheckForUpdates checks if a newer version is available
func (vc *VersionChecker) CheckForUpdates(ctx context.Context) VersionInfo {
	info := VersionInfo{
		CurrentVersion: version,
	}

	// Skip check if we're in dev mode
	if version == "dev" {
		return info
	}

	// Fetch latest version from npm
	latestVersion, err := vc.fetchLatestVersion(ctx)
	if err != nil {
		info.Error = fmt.Errorf("failed to check for updates: %w", err)
		return info
	}

	info.LatestVersion = latestVersion
	info.UpdateAvailable = vc.isUpdateAvailable(version, latestVersion)

	return info
}

// fetchLatestVersion fetches the latest version from npm registry
func (vc *VersionChecker) fetchLatestVersion(ctx context.Context) (string, error) {
	url := "https://registry.npmjs.org/404skill"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := vc.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var npmResp NPMResponse
	if err := json.NewDecoder(resp.Body).Decode(&npmResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return npmResp.DistTags.Latest, nil
}

// isUpdateAvailable compares current and latest versions
func (vc *VersionChecker) isUpdateAvailable(current, latest string) bool {
	// Remove 'v' prefix if present
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")

	// Simple string comparison for now
	// In a production environment, you might want to use proper semver parsing
	return current != latest
}

// GetUpdateMessage returns a formatted update message
func (vc *VersionChecker) GetUpdateMessage(info VersionInfo) string {
	if info.Error != nil {
		return ""
	}

	if !info.UpdateAvailable {
		return ""
	}

	return fmt.Sprintf("Update available: %s → %s", info.CurrentVersion, info.LatestVersion)
}
