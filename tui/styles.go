package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	btable "github.com/evertras/bubble-table/table"
)

// Colors
var (
	primary    = lipgloss.Color("#00ff00") // Bright green
	secondary  = lipgloss.Color("#00aa00") // Darker green
	accent     = lipgloss.Color("#00ffaa") // Cyan-green
	errorColor = lipgloss.Color("#ff0000") // Red
	bg         = lipgloss.Color("#000000") // Black
)

// Styles
var (
	baseStyle = lipgloss.NewStyle().
			Foreground(primary).
			Background(bg).
			Padding(1, 2)

	headerStyle = lipgloss.NewStyle().
			Foreground(accent).
			Bold(true).
			Underline(true).
			Padding(0, 1)

	menuStyle = lipgloss.NewStyle().
			Foreground(primary).
			Background(bg).
			Padding(0, 1)

	menuItemStyle = lipgloss.NewStyle().
			Foreground(primary).
			Background(bg).
			Padding(0, 1)

	selectedMenuItemStyle = lipgloss.NewStyle().
				Foreground(bg).
				Background(primary).
				Bold(true).
				Padding(0, 1)

	loginBoxStyle = lipgloss.NewStyle().
			Foreground(primary).
			Background(bg).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(1, 4).
			Width(44)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(secondary).
			Faint(true)

	asciiArt = lipgloss.NewStyle().
			Foreground(primary).Render(`
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
||                                    Version: ` + version + `                                ||
||                                                                                            ||
\==============================================================================================/
                                                                       `)

	downloadedStyle = lipgloss.NewStyle().
			Foreground(secondary).
			Faint(true).
			Render("✓ Downloaded")

	bubbleTableColumns = []btable.Column{
		btable.NewColumn("name", "Name", 32),
		btable.NewColumn("lang", "Language", 15),
		btable.NewColumn("diff", "Difficulty", 15),
		btable.NewColumn("dur", "Duration", 15),
		btable.NewColumn("status", "Status", 15),
	}

	spinnerStyle = lipgloss.NewStyle().
			Foreground(accent).
			Bold(true).
			Padding(0, 1)
)

// GetASCIIArt returns the ASCII art with version information and update status
func GetASCIIArt(versionInfo VersionInfo) string {
	updateMsg := ""
	if versionInfo.UpdateAvailable {
		updateMsg = fmt.Sprintf("Latest version: %s \t Run 'npm update -g 404skill' to upgrade", versionInfo.LatestVersion)
	}

	return lipgloss.NewStyle().
		Foreground(primary).Render(`
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
