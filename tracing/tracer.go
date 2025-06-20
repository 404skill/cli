// Package tracing provides comprehensive user interaction and performance tracking
// for the 404skill CLI application.
package tracing

import (
	"time"
)

// Tracer defines the contract for tracking user interactions and system events.
// This interface follows the Dependency Inversion Principle, allowing different
// implementations (local, remote, composite, no-op) without changing client code.
type Tracer interface {
	// TrackEvent records a structured event with automatic timestamp and session context
	TrackEvent(event Event) error

	// TrackUserAction records user interactions like key presses, menu selections
	TrackUserAction(action UserActionEvent) error

	// TrackPerformance records timing and performance metrics
	TrackPerformance(metric PerformanceEvent) error

	// TrackNavigation records state transitions and user journey
	TrackNavigation(nav NavigationEvent) error

	// TrackError records errors and diagnostic information
	TrackError(err ErrorEvent) error

	// Flush ensures all pending events are persisted
	Flush() error

	// Close gracefully shuts down the tracer and performs cleanup
	Close() error
}

// Event represents the base interface for all trackable events.
// This follows the Strategy pattern, allowing different event types
// to be processed uniformly while maintaining type safety.
type Event interface {
	// EventType returns the type identifier for this event
	EventType() string

	// Timestamp returns when this event occurred
	Timestamp() time.Time

	// Validate ensures the event data is complete and valid
	Validate() error

	// Sanitize removes or masks any sensitive information
	Sanitize() Event
}

// SessionInfo contains metadata about the current user session
type SessionInfo struct {
	ID        string    `json:"session_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time,omitempty"`
	UserAgent string    `json:"user_agent"`
	Platform  string    `json:"platform"`
	Version   string    `json:"version"`
}

// EventBatch represents a collection of events for batch processing
type EventBatch struct {
	Session SessionInfo `json:"session"`
	Events  []Event     `json:"events"`
}

// TracingConfig holds configuration for the tracing system
type TracingConfig struct {
	Enabled        bool          `json:"enabled"`
	UploadEndpoint string        `json:"upload_endpoint"`
	LocalDir       string        `json:"local_dir"`
	MaxSessions    int           `json:"max_sessions"`
	UploadTimeout  time.Duration `json:"upload_timeout"`
	FlushInterval  time.Duration `json:"flush_interval"`
	MaxBufferSize  int           `json:"max_buffer_size"`
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() TracingConfig {
	return TracingConfig{
		Enabled:        true,
		UploadEndpoint: "https://api.404skill.com/v1/telemetry",
		LocalDir:       "~/.404skill/traces",
		MaxSessions:    10,
		UploadTimeout:  30 * time.Second,
		FlushInterval:  10 * time.Second,
		MaxBufferSize:  1000,
	}
}

// TracerFactory creates tracer instances based on configuration.
// This implements the Factory pattern for clean object creation.
type TracerFactory interface {
	CreateTracer(config TracingConfig) (Tracer, error)
	CreateTracerWithVersion(config TracingConfig, version string) (Tracer, error)
}

// NoOpTracer provides a null object implementation for when tracing is disabled.
// This follows the Null Object pattern to eliminate nil checks in client code.
type NoOpTracer struct{}

func (n *NoOpTracer) TrackEvent(event Event) error                   { return nil }
func (n *NoOpTracer) TrackUserAction(action UserActionEvent) error   { return nil }
func (n *NoOpTracer) TrackPerformance(metric PerformanceEvent) error { return nil }
func (n *NoOpTracer) TrackNavigation(nav NavigationEvent) error      { return nil }
func (n *NoOpTracer) TrackError(err ErrorEvent) error                { return nil }
func (n *NoOpTracer) Flush() error                                   { return nil }
func (n *NoOpTracer) Close() error                                   { return nil }

// NewNoOpTracer creates a tracer that discards all events
func NewNoOpTracer() Tracer {
	return &NoOpTracer{}
}
