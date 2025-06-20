package tracing

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// BaseEvent provides common functionality for all event types.
// This implements the Template Method pattern, providing shared behavior
// while allowing subclasses to customize specific aspects.
type BaseEvent struct {
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"timestamp"`
	SessionID string    `json:"session_id"`
}

// EventType returns the type identifier for this event
func (b BaseEvent) EventType() string {
	return b.Type
}

// Timestamp returns when this event occurred
func (b BaseEvent) Timestamp() time.Time {
	return b.CreatedAt
}

// UserActionEvent tracks user interactions like key presses and menu selections
type UserActionEvent struct {
	BaseEvent
	Action     string            `json:"action"`               // e.g., "key_press", "menu_select", "button_click"
	Target     string            `json:"target"`               // e.g., "main_menu", "project_list", "login_form"
	Key        string            `json:"key,omitempty"`        // For key press events
	Value      string            `json:"value,omitempty"`      // For input events (sanitized)
	Properties map[string]string `json:"properties,omitempty"` // Additional context
}

// NewUserActionEvent creates a new user action event
func NewUserActionEvent(sessionID, action, target string) *UserActionEvent {
	return &UserActionEvent{
		BaseEvent: BaseEvent{
			Type:      "user_action",
			CreatedAt: time.Now(),
			SessionID: sessionID,
		},
		Action:     action,
		Target:     target,
		Properties: make(map[string]string),
	}
}

// Validate ensures the event data is complete and valid
func (u *UserActionEvent) Validate() error {
	if u.Action == "" {
		return errors.New("action is required")
	}
	if u.Target == "" {
		return errors.New("target is required")
	}
	return nil
}

// Sanitize removes or masks any sensitive information
func (u *UserActionEvent) Sanitize() Event {
	sanitized := *u

	// Mask sensitive values
	if strings.Contains(strings.ToLower(u.Target), "password") ||
		strings.Contains(strings.ToLower(u.Target), "token") ||
		strings.Contains(strings.ToLower(u.Target), "secret") {
		sanitized.Value = "[REDACTED]"
	}

	// Remove sensitive properties
	if sanitized.Properties != nil {
		sanitized.Properties = make(map[string]string)
		for k, v := range u.Properties {
			if !isSensitiveKey(k) {
				sanitized.Properties[k] = v
			}
		}
	}

	return &sanitized
}

// Duration wraps time.Duration to provide human-readable JSON serialization
type Duration time.Duration

// MarshalJSON implements json.Marshaler interface
func (d Duration) MarshalJSON() ([]byte, error) {
	duration := time.Duration(d)
	return json.Marshal(map[string]interface{}{
		"nanoseconds":  int64(duration),
		"readable":     duration.String(),
		"milliseconds": duration.Nanoseconds() / 1000000,
	})
}

// UnmarshalJSON implements json.Unmarshaler interface
func (d *Duration) UnmarshalJSON(data []byte) error {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	// Handle different input formats
	switch value := v.(type) {
	case float64:
		*d = Duration(time.Duration(value))
	case map[string]interface{}:
		if ns, ok := value["nanoseconds"].(float64); ok {
			*d = Duration(time.Duration(ns))
		}
	}

	return nil
}

// PerformanceEvent tracks timing and performance metrics
type PerformanceEvent struct {
	BaseEvent
	Operation string            `json:"operation"`          // e.g., "api_call", "file_load", "render"
	Duration  Duration          `json:"duration"`           // How long the operation took (with units)
	Success   bool              `json:"success"`            // Whether the operation succeeded
	Metadata  map[string]string `json:"metadata,omitempty"` // Additional performance context
}

// NewPerformanceEvent creates a new performance event
func NewPerformanceEvent(sessionID, operation string, duration time.Duration, success bool) *PerformanceEvent {
	return &PerformanceEvent{
		BaseEvent: BaseEvent{
			Type:      "performance",
			CreatedAt: time.Now(),
			SessionID: sessionID,
		},
		Operation: operation,
		Duration:  Duration(duration),
		Success:   success,
		Metadata:  make(map[string]string),
	}
}

// Validate ensures the event data is complete and valid
func (p *PerformanceEvent) Validate() error {
	if p.Operation == "" {
		return errors.New("operation is required")
	}
	if time.Duration(p.Duration) < 0 {
		return errors.New("duration cannot be negative")
	}
	return nil
}

// Sanitize removes or masks any sensitive information
func (p *PerformanceEvent) Sanitize() Event {
	sanitized := *p

	// Remove sensitive metadata
	if sanitized.Metadata != nil {
		sanitized.Metadata = make(map[string]string)
		for k, v := range p.Metadata {
			if !isSensitiveKey(k) {
				sanitized.Metadata[k] = v
			}
		}
	}

	return &sanitized
}

// NavigationEvent tracks state transitions and user journey
type NavigationEvent struct {
	BaseEvent
	FromState string            `json:"from_state"`        // Previous state
	ToState   string            `json:"to_state"`          // New state
	Trigger   string            `json:"trigger"`           // What caused the transition
	Context   map[string]string `json:"context,omitempty"` // Additional navigation context
}

// NewNavigationEvent creates a new navigation event
func NewNavigationEvent(sessionID, fromState, toState, trigger string) *NavigationEvent {
	return &NavigationEvent{
		BaseEvent: BaseEvent{
			Type:      "navigation",
			CreatedAt: time.Now(),
			SessionID: sessionID,
		},
		FromState: fromState,
		ToState:   toState,
		Trigger:   trigger,
		Context:   make(map[string]string),
	}
}

// Validate ensures the event data is complete and valid
func (n *NavigationEvent) Validate() error {
	if n.ToState == "" {
		return errors.New("to_state is required")
	}
	if n.Trigger == "" {
		return errors.New("trigger is required")
	}
	return nil
}

// Sanitize removes or masks any sensitive information
func (n *NavigationEvent) Sanitize() Event {
	sanitized := *n

	// Remove sensitive context
	if sanitized.Context != nil {
		sanitized.Context = make(map[string]string)
		for k, v := range n.Context {
			if !isSensitiveKey(k) {
				sanitized.Context[k] = v
			}
		}
	}

	return &sanitized
}

// ErrorEvent tracks errors and diagnostic information
type ErrorEvent struct {
	BaseEvent
	Error     string            `json:"error"`               // Error message (sanitized)
	Code      string            `json:"code,omitempty"`      // Error code if available
	Component string            `json:"component,omitempty"` // Which component generated the error
	Stack     string            `json:"stack,omitempty"`     // Stack trace (sanitized)
	Context   map[string]string `json:"context,omitempty"`   // Additional error context
}

// NewErrorEvent creates a new error event
func NewErrorEvent(sessionID, errorMsg, component string) *ErrorEvent {
	return &ErrorEvent{
		BaseEvent: BaseEvent{
			Type:      "error",
			CreatedAt: time.Now(),
			SessionID: sessionID,
		},
		Error:     errorMsg,
		Component: component,
		Context:   make(map[string]string),
	}
}

// Validate ensures the event data is complete and valid
func (e *ErrorEvent) Validate() error {
	if e.Error == "" {
		return errors.New("error message is required")
	}
	return nil
}

// Sanitize removes or masks any sensitive information
func (e *ErrorEvent) Sanitize() Event {
	sanitized := *e

	// Sanitize error message and stack trace
	sanitized.Error = sanitizeErrorMessage(e.Error)
	sanitized.Stack = sanitizeStackTrace(e.Stack)

	// Remove sensitive context
	if sanitized.Context != nil {
		sanitized.Context = make(map[string]string)
		for k, v := range e.Context {
			if !isSensitiveKey(k) {
				sanitized.Context[k] = v
			}
		}
	}

	return &sanitized
}

// Helper functions for sanitization

// isSensitiveKey checks if a key contains sensitive information
func isSensitiveKey(key string) bool {
	key = strings.ToLower(key)
	sensitiveKeys := []string{
		"password", "token", "secret", "key", "auth", "credential",
		"username", "email", "phone", "address", "ssn", "credit",
	}

	for _, sensitive := range sensitiveKeys {
		if strings.Contains(key, sensitive) {
			return true
		}
	}
	return false
}

// sanitizeErrorMessage removes sensitive information from error messages
func sanitizeErrorMessage(msg string) string {
	// Replace common patterns that might contain sensitive info
	patterns := []string{
		`token=[^&\s]+`,
		`password=[^&\s]+`,
		`key=[^&\s]+`,
		`secret=[^&\s]+`,
		`auth=[^&\s]+`,
	}

	sanitized := msg
	for _, pattern := range patterns {
		// Simple string replacement for common patterns
		if strings.Contains(strings.ToLower(sanitized), strings.Split(pattern, "=")[0]) {
			sanitized = fmt.Sprintf("Error occurred (details redacted for security)")
			break
		}
	}

	return sanitized
}

// sanitizeStackTrace removes sensitive information from stack traces
func sanitizeStackTrace(stack string) string {
	if stack == "" {
		return ""
	}

	// For now, just return a generic message
	// In a real implementation, you might parse and selectively redact
	return "[Stack trace available but redacted for security]"
}
