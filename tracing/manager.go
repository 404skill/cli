package tracing

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

// Manager provides a high-level interface for the tracing system.
// This implements the Facade pattern, hiding the complexity of the
// underlying tracing components and providing a simple API.
type Manager struct {
	tracer    Tracer
	config    TracingConfig
	sessionID string
	mu        sync.RWMutex
	closed    bool
}

// NewManager creates a new tracing manager with the given configuration
func NewManager(config TracingConfig) (*Manager, error) {
	return NewManagerWithVersion(config, "dev")
}

// NewManagerWithVersion creates a new tracing manager with the given configuration and version
func NewManagerWithVersion(config TracingConfig, version string) (*Manager, error) {
	factory := NewDefaultTracerFactory()
	tracer, err := factory.CreateTracerWithVersion(config, version)
	if err != nil {
		return nil, fmt.Errorf("failed to create tracer: %w", err)
	}

	// Extract session ID if possible
	sessionID := "unknown"
	if compositeTracer, ok := tracer.(*CompositeTracer); ok {
		sessionID = compositeTracer.localTracer.session.ID
	}

	manager := &Manager{
		tracer:    tracer,
		config:    config,
		sessionID: sessionID,
	}

	// Track session start
	if err := manager.TrackSessionStart(); err != nil {
		// Don't fail on tracking errors
		_ = err
	}

	return manager, nil
}

// NewDefaultManager creates a manager with default configuration
func NewDefaultManager() (*Manager, error) {
	return NewManager(DefaultConfig())
}

// TrackSessionStart records the beginning of a user session
func (m *Manager) TrackSessionStart() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil
	}

	event := NewNavigationEvent(m.sessionID, "", "session_start", "application_launch")
	event.Context["platform"] = runtime.GOOS
	event.Context["arch"] = runtime.GOARCH
	event.Context["go_version"] = runtime.Version()

	return m.tracer.TrackNavigation(*event)
}

// TrackSessionEnd records the end of a user session
func (m *Manager) TrackSessionEnd() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil
	}

	event := NewNavigationEvent(m.sessionID, "session_active", "session_end", "application_exit")
	return m.tracer.TrackNavigation(*event)
}

// TrackKeyPress records a key press event
func (m *Manager) TrackKeyPress(key, context string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil
	}

	event := NewUserActionEvent(m.sessionID, "key_press", context)
	event.Key = key
	return m.tracer.TrackUserAction(*event)
}

// TrackMenuSelection records a menu selection event
func (m *Manager) TrackMenuSelection(menu, selection string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil
	}

	event := NewUserActionEvent(m.sessionID, "menu_select", menu)
	event.Value = selection
	return m.tracer.TrackUserAction(*event)
}

// TrackStateTransition records a state change in the application
func (m *Manager) TrackStateTransition(fromState, toState, trigger string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil
	}

	event := NewNavigationEvent(m.sessionID, fromState, toState, trigger)
	return m.tracer.TrackNavigation(*event)
}

// TrackOperation records the performance of an operation
func (m *Manager) TrackOperation(operation string, duration time.Duration, success bool) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil
	}

	event := NewPerformanceEvent(m.sessionID, operation, duration, success)
	return m.tracer.TrackPerformance(*event)
}

// TrackOperationWithContext records an operation with additional context
func (m *Manager) TrackOperationWithContext(operation string, duration time.Duration, success bool, metadata map[string]string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil
	}

	event := NewPerformanceEvent(m.sessionID, operation, duration, success)
	for k, v := range metadata {
		event.Metadata[k] = v
	}
	return m.tracer.TrackPerformance(*event)
}

// TrackError records an error event
func (m *Manager) TrackError(err error, component string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil
	}

	event := NewErrorEvent(m.sessionID, err.Error(), component)
	return m.tracer.TrackError(*event)
}

// TrackErrorWithContext records an error with additional context
func (m *Manager) TrackErrorWithContext(err error, component string, context map[string]string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil
	}

	event := NewErrorEvent(m.sessionID, err.Error(), component)
	for k, v := range context {
		event.Context[k] = v
	}
	return m.tracer.TrackError(*event)
}

// TimedOperation provides a convenient way to track operation performance
func (m *Manager) TimedOperation(operation string) *TimedOperationTracker {
	return &TimedOperationTracker{
		manager:   m,
		operation: operation,
		startTime: time.Now(),
	}
}

// Flush ensures all pending events are persisted
func (m *Manager) Flush() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil
	}

	return m.tracer.Flush()
}

// Close gracefully shuts down the tracing system
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	// Track session end directly without calling TrackSessionEnd to avoid deadlock
	event := NewNavigationEvent(m.sessionID, "session_active", "session_end", "application_exit")
	_ = m.tracer.TrackNavigation(*event)

	// Close the tracer
	err := m.tracer.Close()
	m.closed = true

	return err
}

// IsEnabled returns whether tracing is currently enabled
func (m *Manager) IsEnabled() bool {
	return m.config.Enabled
}

// GetSessionID returns the current session ID
func (m *Manager) GetSessionID() string {
	return m.sessionID
}

// TimedOperationTracker helps track the performance of operations
type TimedOperationTracker struct {
	manager   *Manager
	operation string
	startTime time.Time
	metadata  map[string]string
}

// AddMetadata adds metadata to the operation
func (t *TimedOperationTracker) AddMetadata(key, value string) *TimedOperationTracker {
	if t.metadata == nil {
		t.metadata = make(map[string]string)
	}
	t.metadata[key] = value
	return t
}

// Complete marks the operation as completed successfully
func (t *TimedOperationTracker) Complete() error {
	duration := time.Since(t.startTime)
	if t.metadata != nil {
		return t.manager.TrackOperationWithContext(t.operation, duration, true, t.metadata)
	}
	return t.manager.TrackOperation(t.operation, duration, true)
}

// CompleteWithError marks the operation as completed with an error
func (t *TimedOperationTracker) CompleteWithError(err error) error {
	duration := time.Since(t.startTime)

	// Track the performance (as failed)
	var perfErr error
	if t.metadata != nil {
		perfErr = t.manager.TrackOperationWithContext(t.operation, duration, false, t.metadata)
	} else {
		perfErr = t.manager.TrackOperation(t.operation, duration, false)
	}

	// Track the error
	errorContext := map[string]string{
		"operation": t.operation,
		"duration":  duration.String(),
	}
	for k, v := range t.metadata {
		errorContext[k] = v
	}

	errErr := t.manager.TrackErrorWithContext(err, t.operation, errorContext)

	// Return the first error that occurred
	if perfErr != nil {
		return perfErr
	}
	return errErr
}

// Global manager instance for convenience
var globalManager *Manager
var globalManagerOnce sync.Once

// InitGlobalTracing initializes the global tracing manager
func InitGlobalTracing(config TracingConfig) error {
	return InitGlobalTracingWithVersion(config, "dev")
}

// InitGlobalTracingWithVersion initializes the global tracing manager with version
func InitGlobalTracingWithVersion(config TracingConfig, version string) error {
	var err error
	globalManagerOnce.Do(func() {
		globalManager, err = NewManagerWithVersion(config, version)
	})
	return err
}

// InitDefaultGlobalTracing initializes global tracing with default config
func InitDefaultGlobalTracing() error {
	return InitGlobalTracing(DefaultConfig())
}

// GetGlobalManager returns the global tracing manager
func GetGlobalManager() *Manager {
	return globalManager
}

// CloseGlobalTracing closes the global tracing manager
func CloseGlobalTracing() error {
	if globalManager != nil {
		return globalManager.Close()
	}
	return nil
}

// Convenience functions for global tracing

// TrackKeyPress records a key press using the global manager
func TrackKeyPress(key, context string) error {
	if globalManager != nil {
		return globalManager.TrackKeyPress(key, context)
	}
	return nil
}

// TrackMenuSelection records a menu selection using the global manager
func TrackMenuSelection(menu, selection string) error {
	if globalManager != nil {
		return globalManager.TrackMenuSelection(menu, selection)
	}
	return nil
}

// TrackStateTransition records a state transition using the global manager
func TrackStateTransition(fromState, toState, trigger string) error {
	if globalManager != nil {
		return globalManager.TrackStateTransition(fromState, toState, trigger)
	}
	return nil
}

// TrackOperation records an operation using the global manager
func TrackOperation(operation string, duration time.Duration, success bool) error {
	if globalManager != nil {
		return globalManager.TrackOperation(operation, duration, success)
	}
	return nil
}

// TrackError records an error using the global manager
func TrackError(err error, component string) error {
	if globalManager != nil {
		return globalManager.TrackError(err, component)
	}
	return nil
}

// TimedOperation creates a timed operation tracker using the global manager
func TimedOperation(operation string) *TimedOperationTracker {
	if globalManager != nil {
		return globalManager.TimedOperation(operation)
	}
	// Return a no-op tracker if no global manager
	return &TimedOperationTracker{
		operation: operation,
		startTime: time.Now(),
	}
}
