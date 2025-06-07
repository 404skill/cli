package language

import (
	"404skill-cli/api"
	"404skill-cli/downloader"
	"404skill-cli/tui/components/menu"
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Component handles language selection and project downloading
type Component struct {
	// Dependencies
	downloader downloader.Downloader

	// UI components
	menu *menu.Component

	// State
	project     *api.Project
	downloading bool
	progress    float64
	errorMsg    string
	ready       bool
}

// New creates a new language component with dependency injection
func New(project *api.Project, downloader downloader.Downloader) *Component {
	// Extract languages from project
	languages := strings.Split(project.Language, ",")
	for i := range languages {
		languages[i] = strings.TrimSpace(languages[i])
	}

	// Create menu with languages
	languageMenu := menu.New(languages)

	return &Component{
		downloader: downloader,
		menu:       languageMenu,
		project:    project,
	}
}

// SetProject updates the project and rebuilds the language menu
func (c *Component) SetProject(project *api.Project) {
	c.project = project

	// Extract languages and update menu
	languages := strings.Split(project.Language, ",")
	for i := range languages {
		languages[i] = strings.TrimSpace(languages[i])
	}

	c.menu.SetItems(languages)
	c.downloading = false
	c.progress = 0
	c.errorMsg = ""
}

// SetDownloading sets the downloading state
func (c *Component) SetDownloading(downloading bool) {
	c.downloading = downloading
	if !downloading {
		c.progress = 0
	}
}

// SetProgress updates the download progress
func (c *Component) SetProgress(progress float64) {
	c.progress = progress
}

// SetError sets an error message
func (c *Component) SetError(err string) {
	c.errorMsg = err
	c.downloading = false
}

// GetSelectedLanguage returns the currently selected language
func (c *Component) GetSelectedLanguage() string {
	return c.menu.GetSelectedItem()
}

// Update handles component updates
func (c *Component) Update(msg tea.Msg) (*Component, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Pass key messages to menu first
		if !c.downloading {
			menuComponent, menuCmd := c.menu.Update(msg)
			c.menu = menuComponent
			return c, menuCmd
		}
	case menu.MenuSelectMsg:
		if c.project != nil {
			selectedLanguage := msg.SelectedItem
			c.downloading = true
			c.errorMsg = ""
			c.progress = 0

			// Start download asynchronously
			return c, c.startDownload(selectedLanguage)
		}
	case DownloadProgressMsg:
		c.SetProgress(msg.Progress)
		return c, nil
	case DownloadCompleteMsg:
		c.downloading = false
		return c, nil
	case DownloadErrorMsg:
		c.SetError(msg.Error)
		return c, nil
	}

	return c, nil
}

// startDownload initiates the download process
func (c *Component) startDownload(language string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		err := c.downloader.DownloadProject(ctx, c.project, language, nil)
		if err != nil {
			return DownloadErrorMsg{Error: err.Error()}
		}

		return DownloadCompleteMsg{
			Project:  c.project,
			Language: language,
		}
	}
}

// View renders the component
func (c *Component) View() string {
	if c.downloading {
		return c.renderDownloading()
	}

	view := c.renderHeader()
	view += "\n\n" + c.menu.View()
	view += "\n" + c.renderHelp()

	if c.errorMsg != "" {
		view += "\n\n" + c.renderError()
	}

	return view
}

// renderHeader renders the component header
func (c *Component) renderHeader() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ffaa")).
		Bold(true).
		Underline(true).
		Padding(0, 1)

	return style.Render("\nSelect a language for " + c.project.Name)
}

// renderDownloading renders the downloading state with progress
func (c *Component) renderDownloading() string {
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ffaa")).
		Bold(true).
		Underline(true).
		Padding(0, 1)

	progress := int(c.progress * 100)
	progressBar := strings.Repeat("█", progress/10) + strings.Repeat("░", 10-progress/10)

	return fmt.Sprintf("%s\n\nDownloading project...\n[%s] %d%%\n\nPress q to quit",
		headerStyle.Render("Downloading Project"),
		progressBar,
		progress)
}

// renderHelp renders the help text
func (c *Component) renderHelp() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00aa00")).
		Faint(true)

	return style.Render("Use ↑/↓ or k/j to move, Enter to select, [esc/b] back, q to quit")
}

// renderError renders error messages
func (c *Component) renderError() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ff0000")).
		Bold(true)

	return style.Render("Error: " + c.errorMsg)
}
