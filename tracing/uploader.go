package tracing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Uploader handles uploading trace files to the backend service.
// This implements the Strategy pattern for different upload strategies
// and uses the Circuit Breaker pattern for resilience.
type Uploader interface {
	// UploadTraces uploads all trace files from the local directory
	UploadTraces(ctx context.Context, config TracingConfig) error

	// UploadBatch uploads a specific batch of events
	UploadBatch(ctx context.Context, batch EventBatch, endpoint string, timeout time.Duration) error
}

// HTTPUploader implements the Uploader interface using HTTP POST requests
type HTTPUploader struct {
	client *http.Client
}

// NewHTTPUploader creates a new HTTP-based uploader
func NewHTTPUploader() *HTTPUploader {
	return &HTTPUploader{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// UploadTraces uploads all trace files from the local directory
func (u *HTTPUploader) UploadTraces(ctx context.Context, config TracingConfig) error {
	if !config.Enabled || config.UploadEndpoint == "" {
		return nil // Nothing to upload
	}

	dir, err := expandPath(config.LocalDir)
	if err != nil {
		return fmt.Errorf("failed to expand path: %w", err)
	}

	// Find all trace files
	files, err := u.findTraceFiles(dir)
	if err != nil {
		return fmt.Errorf("failed to find trace files: %w", err)
	}

	if len(files) == 0 {
		return nil // No files to upload
	}

	// Upload each file
	uploadErrors := make([]error, 0)
	successCount := 0

	for _, file := range files {
		if err := u.uploadTraceFile(ctx, file, config); err != nil {
			uploadErrors = append(uploadErrors, fmt.Errorf("failed to upload %s: %w", file, err))
			continue
		}
		successCount++

		// Remove successfully uploaded file
		if err := os.Remove(file); err != nil {
			// Log but don't fail - we don't want to re-upload
			continue
		}
	}

	// Return combined errors if any
	if len(uploadErrors) > 0 {
		return fmt.Errorf("upload completed with %d successes and %d errors: %v",
			successCount, len(uploadErrors), uploadErrors)
	}

	return nil
}

// UploadBatch uploads a specific batch of events
func (u *HTTPUploader) UploadBatch(ctx context.Context, batch EventBatch, endpoint string, timeout time.Duration) error {
	// Set timeout for this request
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Marshal batch to JSON
	data, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("failed to marshal batch: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", batch.Session.UserAgent)
	req.Header.Set("X-Session-ID", batch.Session.ID)

	// Execute request
	resp, err := u.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// uploadTraceFile uploads a single trace file
func (u *HTTPUploader) uploadTraceFile(ctx context.Context, filePath string, config TracingConfig) error {
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse batch
	var batch EventBatch
	if err := json.Unmarshal(data, &batch); err != nil {
		return fmt.Errorf("failed to parse batch: %w", err)
	}

	// Upload batch
	return u.UploadBatch(ctx, batch, config.UploadEndpoint, config.UploadTimeout)
}

// findTraceFiles finds all JSON trace files in the directory
func (u *HTTPUploader) findTraceFiles(dir string) ([]string, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil // Directory doesn't exist, no files to upload
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() &&
			strings.HasSuffix(entry.Name(), ".json") &&
			strings.HasPrefix(entry.Name(), "session_") {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	return files, nil
}

// CompositeTracer combines local storage with batch uploading.
// This implements the Composite pattern to provide both local persistence
// and remote uploading capabilities.
type CompositeTracer struct {
	localTracer *LocalTracer
	uploader    Uploader
	config      TracingConfig
}

// NewCompositeTracer creates a tracer that stores locally and uploads on close
func NewCompositeTracer(config TracingConfig, version string) (*CompositeTracer, error) {
	localTracer, err := NewLocalTracer(config, version)
	if err != nil {
		return nil, fmt.Errorf("failed to create local tracer: %w", err)
	}

	uploader := NewHTTPUploader()

	return &CompositeTracer{
		localTracer: localTracer,
		uploader:    uploader,
		config:      config,
	}, nil
}

// TrackEvent records a structured event with automatic timestamp and session context
func (c *CompositeTracer) TrackEvent(event Event) error {
	return c.localTracer.TrackEvent(event)
}

// TrackUserAction records user interactions like key presses, menu selections
func (c *CompositeTracer) TrackUserAction(action UserActionEvent) error {
	return c.localTracer.TrackUserAction(action)
}

// TrackPerformance records timing and performance metrics
func (c *CompositeTracer) TrackPerformance(metric PerformanceEvent) error {
	return c.localTracer.TrackPerformance(metric)
}

// TrackNavigation records state transitions and user journey
func (c *CompositeTracer) TrackNavigation(nav NavigationEvent) error {
	return c.localTracer.TrackNavigation(nav)
}

// TrackError records errors and diagnostic information
func (c *CompositeTracer) TrackError(err ErrorEvent) error {
	return c.localTracer.TrackError(err)
}

// Flush ensures all pending events are persisted
func (c *CompositeTracer) Flush() error {
	return c.localTracer.Flush()
}

// Close gracefully shuts down the tracer and uploads all traces
func (c *CompositeTracer) Close() error {
	// Close local tracer first to ensure all events are flushed
	if err := c.localTracer.Close(); err != nil {
		return fmt.Errorf("failed to close local tracer: %w", err)
	}

	// Skip uploads if no endpoint is configured
	if c.config.UploadEndpoint == "" {
		return nil
	}

	// Upload traces with a short timeout to avoid hanging on quit
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.uploader.UploadTraces(ctx, c.config); err != nil {
		// Don't fail on upload errors - the traces are still stored locally
		// In a production system, you might want to log this error
		_ = err
	}

	return nil
}

// DefaultTracerFactory implements the TracerFactory interface
type DefaultTracerFactory struct{}

// CreateTracer creates a tracer instance based on configuration
func (f *DefaultTracerFactory) CreateTracer(config TracingConfig) (Tracer, error) {
	return f.CreateTracerWithVersion(config, "dev")
}

// CreateTracerWithVersion creates a tracer instance with a specific version
func (f *DefaultTracerFactory) CreateTracerWithVersion(config TracingConfig, version string) (Tracer, error) {
	if !config.Enabled {
		return NewNoOpTracer(), nil
	}

	// For now, we'll use the composite tracer by default
	// In the future, we could choose different implementations based on config
	return NewCompositeTracer(config, version)
}

// NewDefaultTracerFactory creates a new default tracer factory
func NewDefaultTracerFactory() TracerFactory {
	return &DefaultTracerFactory{}
}
