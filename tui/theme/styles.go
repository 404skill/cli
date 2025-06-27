package theme

import (
	"github.com/charmbracelet/lipgloss"
)

// ColorScheme defines colors for a specific theme
type ColorScheme struct {
	Primary    lipgloss.Color
	Secondary  lipgloss.Color
	Accent     lipgloss.Color
	Error      lipgloss.Color
	Background lipgloss.Color
	Text       lipgloss.Color
	Muted      lipgloss.Color
	Success    lipgloss.Color
	Warning    lipgloss.Color
	Info       lipgloss.Color
}

// DarkTheme colors (current default)
var DarkTheme = ColorScheme{
	Primary:    lipgloss.Color("#00ff00"), // Bright green
	Secondary:  lipgloss.Color("#00aa00"), // Darker green
	Accent:     lipgloss.Color("#00ffaa"), // Cyan-green
	Error:      lipgloss.Color("#ff0000"), // Red
	Background: lipgloss.Color("#000000"), // Black
	Text:       lipgloss.Color("#ffffff"), // White
	Muted:      lipgloss.Color("#888888"), // Gray
	Success:    lipgloss.Color("#00aa00"), // Green
	Warning:    lipgloss.Color("#ffaa00"), // Orange
	Info:       lipgloss.Color("#00aaff"), // Blue
}

// LightTheme colors
var LightTheme = ColorScheme{
	Primary:    lipgloss.Color("#006600"), // Dark green
	Secondary:  lipgloss.Color("#008800"), // Medium green
	Accent:     lipgloss.Color("#0066aa"), // Blue-green
	Error:      lipgloss.Color("#cc0000"), // Dark red
	Background: lipgloss.Color("#ffffff"), // White
	Text:       lipgloss.Color("#000000"), // Black
	Muted:      lipgloss.Color("#666666"), // Dark gray
	Success:    lipgloss.Color("#006600"), // Dark green
	Warning:    lipgloss.Color("#cc6600"), // Dark orange
	Info:       lipgloss.Color("#0066cc"), // Dark blue
}

// Manager handles theme-aware styling
type Manager struct {
	detector *Detector
	theme    Theme
	colors   ColorScheme
}

// NewManager creates a new theme manager
func NewManager() *Manager {
	detector := NewDetector()
	theme := detector.DetectTheme()

	var colors ColorScheme
	switch theme {
	case ThemeLight:
		colors = LightTheme
	default:
		colors = DarkTheme
	}

	return &Manager{
		detector: detector,
		theme:    theme,
		colors:   colors,
	}
}

// GetTheme returns the current detected theme
func (m *Manager) GetTheme() Theme {
	return m.theme
}

// GetColors returns the current color scheme
func (m *Manager) GetColors() ColorScheme {
	return m.colors
}

// RefreshTheme re-detects the theme and updates colors
func (m *Manager) RefreshTheme() {
	m.theme = m.detector.DetectTheme()
	switch m.theme {
	case ThemeLight:
		m.colors = LightTheme
	default:
		m.colors = DarkTheme
	}
}

// Common Styles - these replace the hardcoded styles from the original styles.go

// BaseStyle returns the base style with theme-aware colors
func (m *Manager) BaseStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(m.colors.Primary).
		Background(m.colors.Background).
		Padding(1, 2)
}

// HeaderStyle returns the header style with theme-aware colors
func (m *Manager) HeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(m.colors.Accent).
		Bold(true).
		Underline(true).
		Padding(0, 1)
}

// MenuStyle returns the menu style with theme-aware colors
func (m *Manager) MenuStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(m.colors.Primary).
		Background(m.colors.Background).
		Padding(0, 1)
}

// MenuItemStyle returns the menu item style with theme-aware colors
func (m *Manager) MenuItemStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(m.colors.Primary).
		Background(m.colors.Background).
		Padding(0, 1)
}

// SelectedMenuItemStyle returns the selected menu item style with theme-aware colors
func (m *Manager) SelectedMenuItemStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(m.colors.Background).
		Background(m.colors.Primary).
		Bold(true).
		Padding(0, 1)
}

// LoginBoxStyle returns the login box style with theme-aware colors
func (m *Manager) LoginBoxStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(m.colors.Primary).
		Background(m.colors.Background).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.colors.Accent).
		Padding(1, 4).
		Width(44)
}

// ErrorStyle returns the error style with theme-aware colors
func (m *Manager) ErrorStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(m.colors.Error).
		Bold(true)
}

// HelpStyle returns the help style with theme-aware colors
func (m *Manager) HelpStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(m.colors.Secondary).
		Faint(true)
}

// SuccessStyle returns the success style with theme-aware colors
func (m *Manager) SuccessStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(m.colors.Success).
		Bold(true)
}

// WarningStyle returns the warning style with theme-aware colors
func (m *Manager) WarningStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(m.colors.Warning).
		Bold(true)
}

// InfoStyle returns the info style with theme-aware colors
func (m *Manager) InfoStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(m.colors.Info).
		Bold(true)
}

// MutedStyle returns the muted style with theme-aware colors
func (m *Manager) MutedStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(m.colors.Muted).
		Faint(true)
}

// TextStyle returns the text style with theme-aware colors
func (m *Manager) TextStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(m.colors.Text).
		Background(m.colors.Background)
}

// SpinnerStyle returns the spinner style with theme-aware colors
func (m *Manager) SpinnerStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(m.colors.Accent).
		Bold(true).
		Padding(0, 1)
}

// DownloadedStyle returns the downloaded indicator style with theme-aware colors
func (m *Manager) DownloadedStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(m.colors.Success).
		Faint(true)
}

// Table styles for bubble-table
func (m *Manager) TableHeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(m.colors.Accent).
		Bold(true).
		Align(lipgloss.Center)
}

func (m *Manager) TableRowStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(m.colors.Text).
		Background(m.colors.Background)
}

func (m *Manager) TableSelectedRowStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(m.colors.Background).
		Background(m.colors.Primary).
		Bold(true)
}

// Utility function to get a style with custom colors
func (m *Manager) CustomStyle(fg, bg lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(fg).
		Background(bg)
}

// Utility function to get a style with just foreground color
func (m *Manager) ForegroundStyle(fg lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(fg)
}
