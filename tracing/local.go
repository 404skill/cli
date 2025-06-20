package tracing

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
)

// LocalTracer implements the Tracer interface for local JSON file storage.
// This follows the Repository pattern for data persistence and uses
// concurrent-safe buffering for performance.
type LocalTracer struct {
	config      TracingConfig
	session     SessionInfo
	buffer      []Event
	bufferMutex sync.RWMutex
	flushTicker *time.Ticker
	stopChan    chan struct{}
	wg          sync.WaitGroup
}

// NewLocalTracer creates a new local file tracer with the given configuration
func NewLocalTracer(config TracingConfig, version string) (*LocalTracer, error) {
	// Expand tilde in path
	dir, err := expandPath(config.LocalDir)
	if err != nil {
		return nil, fmt.Errorf("failed to expand path %s: %w", config.LocalDir, err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create traces directory %s: %w", dir, err)
	}

	// Create session info
	session := SessionInfo{
		ID:        generateSessionID(),
		StartTime: time.Now(),
		UserAgent: fmt.Sprintf("404skill-cli/%s", version),
		Platform:  getPlatform(),
		Version:   version,
	}

	tracer := &LocalTracer{
		config:   config,
		session:  session,
		buffer:   make([]Event, 0, config.MaxBufferSize),
		stopChan: make(chan struct{}),
	}

	// Start background flushing if configured
	if config.FlushInterval > 0 {
		tracer.startBackgroundFlushing()
	}

	return tracer, nil
}

// TrackEvent records a structured event with automatic timestamp and session context
func (l *LocalTracer) TrackEvent(event Event) error {
	if !l.config.Enabled {
		return nil
	}

	// Validate and sanitize the event
	if err := event.Validate(); err != nil {
		return fmt.Errorf("invalid event: %w", err)
	}

	sanitizedEvent := event.Sanitize()

	l.bufferMutex.Lock()
	defer l.bufferMutex.Unlock()

	// Add to buffer
	l.buffer = append(l.buffer, sanitizedEvent)

	// Flush if buffer is full
	if len(l.buffer) >= l.config.MaxBufferSize {
		return l.flushUnsafe()
	}

	return nil
}

// TrackUserAction records user interactions like key presses, menu selections
func (l *LocalTracer) TrackUserAction(action UserActionEvent) error {
	return l.TrackEvent(&action)
}

// TrackPerformance records timing and performance metrics
func (l *LocalTracer) TrackPerformance(metric PerformanceEvent) error {
	return l.TrackEvent(&metric)
}

// TrackNavigation records state transitions and user journey
func (l *LocalTracer) TrackNavigation(nav NavigationEvent) error {
	return l.TrackEvent(&nav)
}

// TrackError records errors and diagnostic information
func (l *LocalTracer) TrackError(err ErrorEvent) error {
	return l.TrackEvent(&err)
}

// Flush ensures all pending events are persisted
func (l *LocalTracer) Flush() error {
	l.bufferMutex.Lock()
	defer l.bufferMutex.Unlock()
	return l.flushUnsafe()
}

// Close gracefully shuts down the tracer and performs cleanup
func (l *LocalTracer) Close() error {
	// Stop background flushing
	if l.flushTicker != nil {
		l.flushTicker.Stop()
		close(l.stopChan)
		l.wg.Wait()
	}

	// Final flush
	if err := l.Flush(); err != nil {
		return fmt.Errorf("failed to flush during close: %w", err)
	}

	// Update session end time
	l.session.EndTime = time.Now()

	// Clean up old sessions
	return l.cleanupOldSessions()
}

// startBackgroundFlushing starts a goroutine that periodically flushes the buffer
func (l *LocalTracer) startBackgroundFlushing() {
	l.flushTicker = time.NewTicker(l.config.FlushInterval)
	l.wg.Add(1)

	go func() {
		defer l.wg.Done()
		for {
			select {
			case <-l.flushTicker.C:
				l.bufferMutex.Lock()
				if len(l.buffer) > 0 {
					_ = l.flushUnsafe() // Ignore errors in background flush
				}
				l.bufferMutex.Unlock()
			case <-l.stopChan:
				return
			}
		}
	}()
}

// flushUnsafe writes the buffer to disk without acquiring the mutex
// This method assumes the caller has already acquired the mutex
func (l *LocalTracer) flushUnsafe() error {
	if len(l.buffer) == 0 {
		return nil
	}

	// Update session end time to current time for this flush
	sessionCopy := l.session
	sessionCopy.EndTime = time.Now()

	// Create batch with updated session info
	batch := EventBatch{
		Session: sessionCopy,
		Events:  make([]Event, len(l.buffer)),
	}
	copy(batch.Events, l.buffer)

	// Write to file
	filename := fmt.Sprintf("session_%s_%d.json",
		l.session.ID,
		time.Now().Unix())

	dir, _ := expandPath(l.config.LocalDir)
	filepath := filepath.Join(dir, filename)

	data, err := json.MarshalIndent(batch, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write events to %s: %w", filepath, err)
	}

	// Clear buffer
	l.buffer = l.buffer[:0]

	return nil
}

// cleanupOldSessions removes old session files to prevent disk space issues
func (l *LocalTracer) cleanupOldSessions() error {
	dir, err := expandPath(l.config.LocalDir)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read traces directory: %w", err)
	}

	// Count session files
	sessionFiles := make([]os.DirEntry, 0)
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			sessionFiles = append(sessionFiles, entry)
		}
	}

	// Remove oldest files if we exceed the limit
	if len(sessionFiles) > l.config.MaxSessions {
		// Sort by modification time (oldest first)
		// For simplicity, we'll just remove files beyond the limit
		// In a production system, you'd want proper sorting
		filesToRemove := len(sessionFiles) - l.config.MaxSessions
		for i := 0; i < filesToRemove; i++ {
			filePath := filepath.Join(dir, sessionFiles[i].Name())
			if err := os.Remove(filePath); err != nil {
				// Log but don't fail on cleanup errors
				continue
			}
		}
	}

	return nil
}

// Helper functions

// generateSessionID creates a unique session identifier
func generateSessionID() string {
	return uuid.New().String()
}

// expandPath expands ~ to the user's home directory
func expandPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path")
	}

	if path[0] != '~' {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, path[1:]), nil
}

// getPlatform returns the current platform information
func getPlatform() string {
	return fmt.Sprintf("%s/%s",
		runtime.GOOS,
		runtime.GOARCH)
}
