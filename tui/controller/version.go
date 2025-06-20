package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Masterminds/semver/v3"
)

// VersionInfo contains version information
type VersionInfo struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	CheckError      error
}

// VersionChecker handles version checking functionality
type VersionChecker struct {
	currentVersion string
	httpClient     *http.Client
}

// NewVersionChecker creates a new version checker
func NewVersionChecker(currentVersion string) *VersionChecker {
	return &VersionChecker{
		currentVersion: currentVersion,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CheckForUpdates checks if a newer version is available
func (vc *VersionChecker) CheckForUpdates(ctx context.Context) VersionInfo {
	info := VersionInfo{
		CurrentVersion: vc.currentVersion,
	}

	// Get latest version from npm registry
	latestVersion, err := vc.getLatestVersionFromNPM(ctx)
	if err != nil {
		info.CheckError = err
		return info
	}

	info.LatestVersion = latestVersion

	// Compare versions
	current, err := semver.NewVersion(vc.currentVersion)
	if err != nil {
		info.CheckError = fmt.Errorf("invalid current version: %v", err)
		return info
	}

	latest, err := semver.NewVersion(latestVersion)
	if err != nil {
		info.CheckError = fmt.Errorf("invalid latest version: %v", err)
		return info
	}

	info.UpdateAvailable = latest.GreaterThan(current)
	return info
}

// getLatestVersionFromNPM fetches the latest version from npm registry
func (vc *VersionChecker) getLatestVersionFromNPM(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://registry.npmjs.org/404skill/latest", nil)
	if err != nil {
		return "", err
	}

	resp, err := vc.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("npm registry returned status %d", resp.StatusCode)
	}

	var npmResponse struct {
		Version string `json:"version"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&npmResponse); err != nil {
		return "", err
	}

	return npmResponse.Version, nil
}
