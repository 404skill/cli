package styles

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	btable "github.com/evertras/bubble-table/table"
)

// Colors
var (
	Primary    = lipgloss.Color("#00ff00") // Bright green
	Secondary  = lipgloss.Color("#00aa00") // Darker green
	Accent     = lipgloss.Color("#00ffaa") // Cyan-green
	ErrorColor = lipgloss.Color("#ff0000") // Red
	Background = lipgloss.Color("#000000") // Black
)

// Common Styles
var (
	BaseStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Background(Background).
			Padding(1, 2)

	HeaderStyle = lipgloss.NewStyle().
			Foreground(Accent).
			Bold(true).
			Underline(true).
			Padding(0, 1)

	MenuStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Background(Background).
			Padding(0, 1)

	MenuItemStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Background(Background).
			Padding(0, 1)

	SelectedMenuItemStyle = lipgloss.NewStyle().
				Foreground(Background).
				Background(Primary).
				Bold(true).
				Padding(0, 1)

	LoginBoxStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Background(Background).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Accent).
			Padding(1, 4).
			Width(44)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ErrorColor).
			Bold(true)

	HelpStyle = lipgloss.NewStyle().
			Foreground(Secondary).
			Faint(true)

	DownloadedStyle = lipgloss.NewStyle().
			Foreground(Secondary).
			Faint(true).
			Render("✓ Downloaded")

	SpinnerStyle = lipgloss.NewStyle().
			Foreground(Accent).
			Bold(true).
			Padding(0, 1)
)

// Table Configuration
var (
	BubbleTableColumns = []btable.Column{
		btable.NewColumn("name", "Name", 32),
		btable.NewColumn("lang", "Language", 15),
		btable.NewColumn("diff", "Difficulty", 15),
		btable.NewColumn("dur", "Duration", 15),
		btable.NewColumn("status", "Status", 15),
	}
)

// VersionInfo represents version information for display
type VersionInfo struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	CheckError      error
}

// GetASCIIArt returns the ASCII art with version information and update status
func GetASCIIArt(versionInfo VersionInfo) string {
	updateMsg := ""
	if versionInfo.UpdateAvailable {
		updateMsg = fmt.Sprintf("Latest version: %s \t Run 'npm update -g 404skill' to upgrade", versionInfo.LatestVersion)
	}

	return lipgloss.NewStyle().
		Foreground(Primary).Render(`
/==============================================================================================\
||                                                                                            ||
||      ___   ___  ________  ___   ___  ________  ___  __    ___  ___       ___               ||
||     |\  \ |\  \|\   __  \|\  \ |\  \|\   ____\|\  \|\  \ |\  \|\  \     |\  \              ||
||     \ \  \\_\  \ \  \|\  \ \  \\_\  \ \  \___|\ \  \/  /|\ \  \ \  \    \ \  \             ||
||      \ \______  \ \  \\\  \ \______  \ \_____  \ \   ___  \ \  \ \  \    \ \  \            ||
||       \|_____|\  \ \  \\\  \|_____|\  \|____|\  \ \  \\ \  \ \  \ \  \____\ \  \____       ||
||              \ \__\ \_______\     \ \__\____\_\  \ \__\\ \__\ \__\ \_______\ \_______\     ||
||               \|__|\|_______|      \|__|\_________\|__| \|__|\|__|\|_______|\|_______|     ||
||                                        \|_________|                                        ||
||                                                                                            ||
\==============================================================================================/
                                                                        
Version: ` + versionInfo.CurrentVersion + `

` + updateMsg + `

`)
}
