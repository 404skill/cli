package language

import (
	"404skill-cli/api"
	"404skill-cli/downloader"
	"404skill-cli/tui/components/menu"
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

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

	// Progress state
	currentOperation string

	// Atomic progress for real-time updates
	atomicProgress uint64
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
	atomic.StoreUint64(&c.atomicProgress, uint64(progress*100))
}

// SetCurrentOperation updates the current operation being performed
func (c *Component) SetCurrentOperation(operation string) {
	c.currentOperation = operation
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

// GetAtomicProgress returns the current atomic progress
func (c *Component) GetAtomicProgress() float64 {
	return float64(atomic.LoadUint64(&c.atomicProgress)) / 100.0
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
			c.currentOperation = "Starting download..."

			// Start download with progress updates
			return c, c.downloadWithProgress(selectedLanguage)
		}
	case DownloadProgressMsg:
		c.SetProgress(msg.Progress)
		return c, c.progressTicker() // Continue progress updates
	case DownloadCompleteMsg:
		c.downloading = false
		c.progress = 1.0
		c.currentOperation = "Download complete!"
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

		// Create progress callback that updates atomic progress
		progressCallback := func(progress float64) {
			// Update atomic progress for real-time updates
			atomic.StoreUint64(&c.atomicProgress, uint64(progress*100))
		}

		// Set initial operation
		c.SetCurrentOperation("Preparing download...")

		err := c.downloader.DownloadProject(ctx, c.project, language, progressCallback)
		if err != nil {
			return DownloadErrorMsg{Error: err.Error()}
		}

		return DownloadCompleteMsg{
			Project:  c.project,
			Language: language,
		}
	}
}

// progressTicker creates a command that sends progress updates
func (c *Component) progressTicker() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
		// Send current atomic progress
		return DownloadProgressMsg{Progress: c.GetAtomicProgress()}
	})
}

// downloadWithProgress creates a command that downloads with progress updates
func (c *Component) downloadWithProgress(language string) tea.Cmd {
	return tea.Batch(
		c.startDownload(language),
		c.progressTicker(),
	)
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

	operation := c.currentOperation
	if operation == "" {
		operation = "Downloading project..."
	}

	return fmt.Sprintf("%s\n\n%s\n[%s] %d%%\n\nPress q to quit",
		headerStyle.Render("Downloading Project"),
		operation,
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
