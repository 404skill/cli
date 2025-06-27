package theme

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Theme represents the detected terminal theme
type Theme int

const (
	ThemeUnknown Theme = iota
	ThemeLight
	ThemeDark
)

// Detector handles terminal theme detection
type Detector struct{}

// NewDetector creates a new theme detector
func NewDetector() *Detector {
	return &Detector{}
}

// DetectTheme detects the current terminal theme using multiple methods
func (d *Detector) DetectTheme() Theme {
	// Try multiple detection methods in order of reliability
	if theme := d.detectFromEnvironment(); theme != ThemeUnknown {
		return theme
	}

	if theme := d.detectFromTerminal(); theme != ThemeUnknown {
		return theme
	}

	if theme := d.detectFromSystem(); theme != ThemeUnknown {
		return theme
	}

	// Default to dark theme for unknown cases
	return ThemeDark
}

// detectFromEnvironment checks environment variables for theme hints
func (d *Detector) detectFromEnvironment() Theme {
	// Check COLORFGBG (common in many terminals)
	if colorfgbg := os.Getenv("COLORFGBG"); colorfgbg != "" {
		// Format is typically "15;0" (light) or "0;15" (dark)
		// First number is foreground, second is background
		parts := strings.Split(colorfgbg, ";")
		if len(parts) >= 2 {
			fg := strings.TrimSpace(parts[0])
			bg := strings.TrimSpace(parts[1])

			// If background is light (high number) and foreground is dark (low number)
			if (bg == "15" || bg == "7") && (fg == "0" || fg == "8") {
				return ThemeLight
			}
			// If background is dark (low number) and foreground is light (high number)
			if (bg == "0" || bg == "8") && (fg == "15" || fg == "7") {
				return ThemeDark
			}
		}
	}

	// Check for specific terminal themes
	if os.Getenv("TERM_PROGRAM") == "iTerm.app" {
		// iTerm2 specific detection
		if os.Getenv("ITERM_PROFILE") != "" {
			// Could check specific profile names that indicate theme
			// For now, default to dark for iTerm2
			return ThemeDark
		}
	}

	// Check for VS Code terminal
	if os.Getenv("TERM_PROGRAM") == "vscode" {
		// VS Code terminal - could check workspace settings
		// For now, default to dark
		return ThemeDark
	}

	return ThemeUnknown
}

// detectFromTerminal attempts to query the terminal directly
func (d *Detector) detectFromTerminal() Theme {
	// Try to query terminal colors using escape sequences
	// This is more reliable but not supported by all terminals

	// Check if we can query terminal colors
	if !d.canQueryTerminal() {
		return ThemeUnknown
	}

	// Try to get background color
	bgColor := d.queryBackgroundColor()
	if bgColor != "" {
		// Parse the color and determine if it's light or dark
		return d.parseColorToTheme(bgColor)
	}

	return ThemeUnknown
}

// detectFromSystem checks system-level theme settings
func (d *Detector) detectFromSystem() Theme {
	switch runtime.GOOS {
	case "darwin":
		return d.detectMacOSTheme()
	case "windows":
		return d.detectWindowsTheme()
	case "linux":
		return d.detectLinuxTheme()
	default:
		return ThemeUnknown
	}
}

// detectMacOSTheme detects theme on macOS
func (d *Detector) detectMacOSTheme() Theme {
	// Use defaults command to check system appearance
	cmd := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle")
	output, err := cmd.Output()
	if err != nil {
		return ThemeUnknown
	}

	style := strings.TrimSpace(string(output))
	if style == "Dark" {
		return ThemeDark
	} else if style == "Light" {
		return ThemeLight
	}

	return ThemeUnknown
}

// detectWindowsTheme detects theme on Windows
func (d *Detector) detectWindowsTheme() Theme {
	// Check Windows registry for theme setting
	cmd := exec.Command("reg", "query", "HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Themes\\Personalize", "/v", "AppsUseLightTheme")
	output, err := cmd.Output()
	if err != nil {
		return ThemeUnknown
	}

	outputStr := strings.ToLower(string(output))
	if strings.Contains(outputStr, "0x00000001") {
		return ThemeLight
	} else if strings.Contains(outputStr, "0x00000000") {
		return ThemeDark
	}

	return ThemeUnknown
}

// detectLinuxTheme detects theme on Linux
func (d *Detector) detectLinuxTheme() Theme {
	// Try multiple methods for Linux

	// Check GTK theme
	if theme := d.detectGTKTheme(); theme != ThemeUnknown {
		return theme
	}

	// Check KDE theme
	if theme := d.detectKDETheme(); theme != ThemeUnknown {
		return theme
	}

	// Check for common environment variables
	if os.Getenv("GTK_THEME") != "" {
		gtkTheme := strings.ToLower(os.Getenv("GTK_THEME"))
		if strings.Contains(gtkTheme, "dark") {
			return ThemeDark
		} else if strings.Contains(gtkTheme, "light") {
			return ThemeLight
		}
	}

	return ThemeUnknown
}

// detectGTKTheme detects GTK theme on Linux
func (d *Detector) detectGTKTheme() Theme {
	// Try gsettings for GNOME
	cmd := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "color-scheme")
	output, err := cmd.Output()
	if err == nil {
		scheme := strings.TrimSpace(string(output))
		if strings.Contains(scheme, "dark") {
			return ThemeDark
		} else if strings.Contains(scheme, "light") {
			return ThemeLight
		}
	}

	return ThemeUnknown
}

// detectKDETheme detects KDE theme on Linux
func (d *Detector) detectKDETheme() Theme {
	// Check KDE configuration
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ThemeUnknown
	}

	kdeConfig := fmt.Sprintf("%s/.config/kdeglobals", homeDir)
	if _, err := os.Stat(kdeConfig); err == nil {
		// Could parse KDE config file for theme info
		// For now, return unknown
		return ThemeUnknown
	}

	return ThemeUnknown
}

// canQueryTerminal checks if the terminal supports color queries
func (d *Detector) canQueryTerminal() bool {
	// Check if we have a TTY and if it supports colors
	if os.Getenv("TERM") == "" {
		return false
	}

	// Check if NO_COLOR is set (indicates no color support)
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	return true
}

// queryBackgroundColor attempts to query the terminal background color
func (d *Detector) queryBackgroundColor() string {
	// This would require implementing terminal escape sequences
	// For now, return empty string
	return ""
}

// parseColorToTheme converts a color string to a theme
func (d *Detector) parseColorToTheme(color string) Theme {
	// This would parse color values and determine if they're light or dark
	// For now, return unknown
	return ThemeUnknown
}

// String returns a string representation of the theme
func (t Theme) String() string {
	switch t {
	case ThemeLight:
		return "light"
	case ThemeDark:
		return "dark"
	default:
		return "unknown"
	}
}
