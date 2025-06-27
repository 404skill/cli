package theme

import (
	"os"
	"testing"
)

func TestDetector_DetectTheme(t *testing.T) {
	detector := NewDetector()
	theme := detector.DetectTheme()

	// Should return either light, dark, or unknown
	if theme != ThemeLight && theme != ThemeDark && theme != ThemeUnknown {
		t.Errorf("Expected theme to be one of ThemeLight, ThemeDark, or ThemeUnknown, got %v", theme)
	}
}

func TestDetector_DetectFromEnvironment(t *testing.T) {
	detector := NewDetector()

	// Test light theme detection
	os.Setenv("COLORFGBG", "0;15")
	theme := detector.DetectTheme()
	if theme != ThemeLight {
		t.Errorf("Expected ThemeLight for COLORFGBG=0;15, got %v", theme)
	}

	// Test dark theme detection
	os.Setenv("COLORFGBG", "15;0")
	theme = detector.DetectTheme()
	if theme != ThemeDark {
		t.Errorf("Expected ThemeDark for COLORFGBG=15;0, got %v", theme)
	}

	// Test unknown theme - should default to dark for unrecognized values
	os.Setenv("COLORFGBG", "7;8")
	theme = detector.DetectTheme()
	if theme != ThemeDark {
		t.Errorf("Expected ThemeDark (default) for COLORFGBG=7;8, got %v", theme)
	}

	// Clean up
	os.Unsetenv("COLORFGBG")
}

func TestDetector_CanQueryTerminal(t *testing.T) {
	detector := NewDetector()

	// Test with TERM set
	os.Setenv("TERM", "xterm-256color")
	canQuery := detector.canQueryTerminal()
	if !canQuery {
		t.Error("Expected canQueryTerminal to return true when TERM is set")
	}

	// Test with NO_COLOR set
	os.Setenv("NO_COLOR", "1")
	canQuery = detector.canQueryTerminal()
	if canQuery {
		t.Error("Expected canQueryTerminal to return false when NO_COLOR is set")
	}

	// Clean up
	os.Unsetenv("TERM")
	os.Unsetenv("NO_COLOR")
}

func TestTheme_String(t *testing.T) {
	tests := []struct {
		theme  Theme
		expect string
	}{
		{ThemeLight, "light"},
		{ThemeDark, "dark"},
		{ThemeUnknown, "unknown"},
	}

	for _, tt := range tests {
		result := tt.theme.String()
		if result != tt.expect {
			t.Errorf("Theme.String() for %v = %s, want %s", tt.theme, result, tt.expect)
		}
	}
}

func TestManager_NewManager(t *testing.T) {
	manager := NewManager()

	// Should have a valid theme
	theme := manager.GetTheme()
	if theme != ThemeLight && theme != ThemeDark && theme != ThemeUnknown {
		t.Errorf("Expected valid theme, got %v", theme)
	}

	// Should have valid colors
	colors := manager.GetColors()
	if colors.Primary == "" {
		t.Error("Expected non-empty primary color")
	}
}

func TestManager_RefreshTheme(t *testing.T) {
	manager := NewManager()

	// Refresh should work without error
	manager.RefreshTheme()

	// Theme should still be valid after refresh
	newTheme := manager.GetTheme()
	if newTheme != ThemeLight && newTheme != ThemeDark && newTheme != ThemeUnknown {
		t.Errorf("Expected valid theme after refresh, got %v", newTheme)
	}
}
