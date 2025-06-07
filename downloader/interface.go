package downloader

import (
	"404skill-cli/api"
	"context"
)

// ProgressCallback is called during download operations to report progress
type ProgressCallback func(progress float64)

// Downloader defines the interface for downloading projects
type Downloader interface {
	// DownloadProject downloads a project in the specified language
	// Returns a channel that will receive progress updates and final result
	DownloadProject(ctx context.Context, project *api.Project, language string, progressCallback ProgressCallback) error
}

// DownloadResult represents the result of a download operation
type DownloadResult struct {
	Success   bool
	Error     error
	Directory string
}
