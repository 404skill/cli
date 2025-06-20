package tracing

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestTracingSystemIntegration demonstrates the complete tracing system
func TestTracingSystemIntegration(t *testing.T) {
	// Create a temporary directory for test traces
	tempDir := t.TempDir()

	// Configure tracing to use temp directory
	config := TracingConfig{
		Enabled:        true,
		UploadEndpoint: "https://test.404skill.com/v1/telemetry",
		LocalDir:       tempDir,
		MaxSessions:    5,
		UploadTimeout:  5 * time.Second,
		FlushInterval:  1 * time.Second,
		MaxBufferSize:  10,
	}

	// Create manager
	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	// Create TUI integration
	integration := NewTUIIntegration(manager)

	// Test various tracking scenarios
	t.Run("UserActions", func(t *testing.T) {
		// Track key presses
		err := manager.TrackKeyPress("enter", "main_menu")
		if err != nil {
			t.Errorf("Failed to track key press: %v", err)
		}

		// Track menu selections
		err = manager.TrackMenuSelection("main_menu", "download_project")
		if err != nil {
			t.Errorf("Failed to track menu selection: %v", err)
		}
	})

	t.Run("StateTransitions", func(t *testing.T) {
		// Track state transitions
		err := manager.TrackStateTransition("main_menu", "project_list", "user_selection")
		if err != nil {
			t.Errorf("Failed to track state transition: %v", err)
		}
	})

	t.Run("PerformanceTracking", func(t *testing.T) {
		// Track timed operations
		tracker := manager.TimedOperation("download_project")
		tracker.AddMetadata("project_name", "react-starter")
		tracker.AddMetadata("language", "typescript")

		// Simulate some work
		time.Sleep(10 * time.Millisecond)

		err := tracker.Complete()
		if err != nil {
			t.Errorf("Failed to complete operation tracking: %v", err)
		}
	})

	t.Run("ErrorTracking", func(t *testing.T) {
		// Track errors
		testError := fmt.Errorf("test error for tracing")
		err := manager.TrackError(testError, "test_component")
		if err != nil {
			t.Errorf("Failed to track error: %v", err)
		}
	})

	t.Run("TUIIntegration", func(t *testing.T) {
		// Test TUI integration helpers
		err := integration.TrackStateChange("login", "main_menu", "authentication_success")
		if err != nil {
			t.Errorf("Failed to track state change via integration: %v", err)
		}

		// Test project operations
		projectTracker := integration.TrackProjectOperation("test", "sample-project")
		projectTracker.AddMetadata("difficulty", "beginner")
		err = projectTracker.Complete()
		if err != nil {
			t.Errorf("Failed to complete project operation: %v", err)
		}
	})

	// Force flush to ensure all events are written
	err = manager.Flush()
	if err != nil {
		t.Errorf("Failed to flush events: %v", err)
	}

	// Verify files were created
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read temp directory: %v", err)
	}

	if len(entries) == 0 {
		t.Error("No trace files were created")
	}

	// Verify file contents
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			filePath := filepath.Join(tempDir, entry.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				t.Errorf("Failed to read trace file %s: %v", entry.Name(), err)
				continue
			}

			if len(data) == 0 {
				t.Errorf("Trace file %s is empty", entry.Name())
			}

			t.Logf("Created trace file: %s (%d bytes)", entry.Name(), len(data))
		}
	}
}

// TestEventSanitization verifies that sensitive data is properly sanitized
func TestEventSanitization(t *testing.T) {
	// Test user action event sanitization
	event := NewUserActionEvent("test-session", "input", "password_field")
	event.Value = "secret123"
	event.Properties["password"] = "another_secret"
	event.Properties["safe_data"] = "this_is_ok"

	sanitized := event.Sanitize().(*UserActionEvent)

	if sanitized.Value != "[REDACTED]" {
		t.Errorf("Expected password value to be redacted, got: %s", sanitized.Value)
	}

	if _, exists := sanitized.Properties["password"]; exists {
		t.Error("Expected password property to be removed")
	}

	if sanitized.Properties["safe_data"] != "this_is_ok" {
		t.Error("Expected safe data to be preserved")
	}
}

// TestNoOpTracer verifies that the no-op tracer works correctly
func TestNoOpTracer(t *testing.T) {
	tracer := NewNoOpTracer()

	// All operations should succeed without error
	err := tracer.TrackEvent(NewUserActionEvent("test", "action", "target"))
	if err != nil {
		t.Errorf("NoOpTracer should not return errors: %v", err)
	}

	err = tracer.Flush()
	if err != nil {
		t.Errorf("NoOpTracer flush should not return errors: %v", err)
	}

	err = tracer.Close()
	if err != nil {
		t.Errorf("NoOpTracer close should not return errors: %v", err)
	}
}

// BenchmarkEventCreation benchmarks event creation performance
func BenchmarkEventCreation(b *testing.B) {
	sessionID := "benchmark-session"

	b.Run("UserActionEvent", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			event := NewUserActionEvent(sessionID, "key_press", "main_menu")
			event.Key = "enter"
			_ = event.Validate()
			_ = event.Sanitize()
		}
	})

	b.Run("PerformanceEvent", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			event := NewPerformanceEvent(sessionID, "api_call", 100*time.Millisecond, true)
			event.Metadata["endpoint"] = "/api/projects"
			_ = event.Validate()
			_ = event.Sanitize()
		}
	})

	b.Run("NavigationEvent", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			event := NewNavigationEvent(sessionID, "main_menu", "project_list", "user_action")
			event.Context["project_count"] = "10"
			_ = event.Validate()
			_ = event.Sanitize()
		}
	})
}
