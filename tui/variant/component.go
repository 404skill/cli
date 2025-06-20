package variant

import (
	"404skill-cli/api"
	"404skill-cli/config"
	"404skill-cli/downloader"
	"404skill-cli/filesystem"
	"404skill-cli/testrunner"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	btable "github.com/evertras/bubble-table/table"
)

// Mode defines the behavior of the variant component
type Mode int

const (
	DownloadMode Mode = iota
	TestMode
)

type Component struct {
	variants         []api.Project
	configManager    *config.ConfigManager
	fileManager      *filesystem.Manager
	downloader       downloader.Downloader
	testRunner       testrunner.TestRunner
	table            btable.Model
	selectedIdx      int
	downloading      bool
	testing          bool
	progress         float64
	errorMsg         string
	infoMsg          string
	ready            bool
	atomicProgress   uint64
	currentOperation string
	selectedVariant  *api.Project
	mode             Mode
	spinnerFrame     string
	outputBuffer     []string
}

func New(variants []api.Project, downloader downloader.Downloader, configManager *config.ConfigManager, fileManager *filesystem.Manager) *Component {
	return NewWithMode(variants, downloader, nil, configManager, fileManager, DownloadMode)
}

func NewForTesting(variants []api.Project, testRunner testrunner.TestRunner, configManager *config.ConfigManager, fileManager *filesystem.Manager) *Component {
	return NewWithMode(variants, nil, testRunner, configManager, fileManager, TestMode)
}

func NewWithMode(variants []api.Project, downloader downloader.Downloader, testRunner testrunner.TestRunner, configManager *config.ConfigManager, fileManager *filesystem.Manager, mode Mode) *Component {
	columns := []btable.Column{
		btable.NewColumn("desc", "Description", 32),
		btable.NewColumn("tech", "Technologies", 24),
		btable.NewColumn("diff", "Difficulty", 12),
	}
	var rows []btable.Row
	for _, v := range variants {
		rows = append(rows, btable.NewRow(map[string]interface{}{
			"desc": v.Description,
			"tech": v.Technologies,
			"diff": v.Difficulty,
		}))
	}
	table := btable.New(columns).WithRows(rows).Focused(true)
	return &Component{
		variants:      variants,
		configManager: configManager,
		fileManager:   fileManager,
		downloader:    downloader,
		testRunner:    testRunner,
		table:         table,
		selectedIdx:   0,
		mode:          mode,
	}
}

func (c *Component) SetDownloading(downloading bool) {
	c.downloading = downloading
	if !downloading {
		c.progress = 0
	}
}

func (c *Component) SetTesting(testing bool) {
	c.testing = testing
	if !testing {
		c.progress = 0
	}
}

func (c *Component) SetProgress(progress float64) {
	c.progress = progress
	atomic.StoreUint64(&c.atomicProgress, uint64(progress*100))
}

func (c *Component) GetAtomicProgress() float64 {
	return float64(atomic.LoadUint64(&c.atomicProgress)) / 100.0
}

func (c *Component) Update(msg tea.Msg) (*Component, tea.Cmd) {
	if c.downloading {
		switch msg := msg.(type) {
		case DownloadProgressMsg:
			c.SetProgress(msg.Progress)
			return c, c.progressTicker()
		case DownloadCompleteMsg:
			c.downloading = false
			c.selectedVariant = msg.Variant
			return c, nil
		case DownloadErrorMsg:
			c.downloading = false
			c.errorMsg = msg.Error
			return c, nil
		}
		return c, c.progressTicker()
	}

	if c.testing {
		switch msg := msg.(type) {
		case TestCompleteMsg:
			c.testing = false
			c.selectedVariant = msg.Variant
			return c, nil
		case TestErrorMsg:
			c.testing = false
			c.errorMsg = msg.Error
			return c, nil
		case spinnerMsg:
			c.spinnerFrame = msg.frame
			return c, c.spinnerTick()
		}
		return c, c.spinnerTick()
	}

	c.table, _ = c.table.Update(msg)

	if m, ok := msg.(tea.KeyMsg); ok {
		switch m.String() {
		case "up", "k":
			if c.selectedIdx > 0 {
				c.selectedIdx--
			}
		case "down", "j":
			if c.selectedIdx < len(c.variants)-1 {
				c.selectedIdx++
			}
		case "enter":
			if c.selectedIdx >= 0 && c.selectedIdx < len(c.variants) {
				variant := c.variants[c.selectedIdx]
				if c.mode == DownloadMode {
					return c.handleDownloadAction(&variant)
				} else {
					return c.handleTestAction(&variant)
				}
			}
		case "esc", "b":
			return c, func() tea.Msg { return BackMsg{} }
		case "q", "ctrl+c":
			return c, func() tea.Msg { return QuitMsg{} }
		}
	}
	return c, nil
}

func (c *Component) handleDownloadAction(variant *api.Project) (*Component, tea.Cmd) {
	if c.configManager != nil && c.configManager.IsProjectDownloaded(variant.ID) {
		if c.fileManager != nil {
			homeDir, err := os.UserHomeDir()
			if err == nil {
				repoName := strings.ToLower(strings.ReplaceAll(variant.Name, " ", "_"))
				projectsDir := filepath.Join(homeDir, "404skill_projects")
				entries, err := os.ReadDir(projectsDir)
				if err == nil {
					var projectDir string
					for _, entry := range entries {
						if entry.IsDir() && strings.HasPrefix(entry.Name(), repoName) {
							projectDir = filepath.Join(projectsDir, entry.Name())
							break
						}
					}
					if projectDir != "" {
						_ = c.fileManager.OpenFileExplorer(projectDir)
					}
				}
			}
		}
		c.infoMsg = "Project already downloaded. Opening project directory..."
		return c, nil
	}
	return c, c.downloadWithProgress(variant)
}

func (c *Component) handleTestAction(variant *api.Project) (*Component, tea.Cmd) {
	// Check if project is downloaded
	if c.configManager == nil || !c.configManager.IsProjectDownloaded(variant.ID) {
		c.errorMsg = "Project must be downloaded before testing. Please download it first."
		return c, nil
	}

	// Start testing with spinner
	c.testing = true
	c.currentOperation = "Initializing tests..."
	c.spinnerFrame = spinnerFrames[0]
	c.outputBuffer = []string{} // Clear previous output
	c.errorMsg = ""             // Clear previous errors
	c.infoMsg = ""              // Clear previous info
	return c, tea.Batch(
		c.startTest(variant),
		c.spinnerTick(),
	)
}

func (c *Component) downloadWithProgress(variant *api.Project) tea.Cmd {
	return tea.Batch(
		c.startDownload(variant),
		c.progressTicker(),
	)
}

func (c *Component) testWithProgress(variant *api.Project) tea.Cmd {
	return tea.Batch(
		c.startTest(variant),
		c.progressTicker(),
	)
}

func (c *Component) startDownload(variant *api.Project) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		progressCallback := func(progress float64) {
			atomic.StoreUint64(&c.atomicProgress, uint64(progress*100))
		}
		c.SetDownloading(true)
		c.currentOperation = "Cloning project..."
		err := c.downloader.DownloadProject(ctx, variant, variant.Language, progressCallback)
		if err != nil {
			return DownloadErrorMsg{Error: err.Error()}
		}
		return DownloadCompleteMsg{Variant: variant}
	}
}

func (c *Component) startTest(variant *api.Project) tea.Cmd {
	return func() tea.Msg {
		// Convert api.Project to testrunner.Project
		testProject := testrunner.Project{
			ID:       variant.ID,
			Name:     variant.Name,
			Language: variant.Language,
		}

		// Progress callback for test runner - update component state
		progressCallback := func(message string) {
			c.outputBuffer = append(c.outputBuffer, message)
			c.currentOperation = message
			// Keep only last 10 messages to prevent memory issues
			if len(c.outputBuffer) > 10 {
				c.outputBuffer = c.outputBuffer[len(c.outputBuffer)-10:]
			}
		}

		// Run tests
		result, err := c.testRunner.RunTests(testProject, progressCallback)
		if err != nil {
			return TestErrorMsg{Error: err.Error()}
		}

		return TestCompleteMsg{Variant: variant, Result: result}
	}
}

func (c *Component) progressTicker() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
		return DownloadProgressMsg{Progress: c.GetAtomicProgress()}
	})
}

func (c *Component) spinnerTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
		idx := 0
		for i, f := range spinnerFrames {
			if f == c.spinnerFrame {
				idx = i
				break
			}
		}
		return spinnerMsg{spinnerFrames[(idx+1)%len(spinnerFrames)]}
	})
}

func (c *Component) View() string {
	if c.downloading {
		return c.renderProgress()
	}

	if c.testing {
		return c.renderTestingSpinner()
	}

	view := c.renderHeader()
	view += "\n\n" + c.renderTable()
	if c.infoMsg != "" {
		view += "\n\n" + c.renderInfo()
	}
	if c.errorMsg != "" {
		view += "\n\n" + c.renderError()
	}
	return view
}

func (c *Component) renderHeader() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ffaa")).
		Bold(true).
		Underline(true).
		Padding(0, 1)

	var headerText string
	if c.mode == DownloadMode {
		headerText = "Select a variant to download:"
	} else {
		headerText = "Select a variant to test:"
	}

	return style.Render(headerText)
}

func (c *Component) renderTable() string {
	return c.table.WithHighlightedRow(c.selectedIdx).View()
}

func (c *Component) renderProgress() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ffaa")).
		Bold(true).
		Padding(0, 1)
	progress := fmt.Sprintf("Progress: %.0f%%", c.progress*100)
	return style.Render(c.currentOperation + "\n" + progress)
}

func (c *Component) renderTestingSpinner() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ffaa")).
		Bold(true).
		Padding(0, 1)

	spinnerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffaa00")).
		Bold(true)

	outputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffffff")).
		Padding(0, 1)

	// Show recent output from test runner
	output := ""
	if len(c.outputBuffer) > 0 {
		// Show last 8 lines of output for better visibility
		start := 0
		if len(c.outputBuffer) > 8 {
			start = len(c.outputBuffer) - 8
		}
		outputLines := c.outputBuffer[start:]
		output = "\n" + outputStyle.Render(strings.Join(outputLines, "\n"))
	}

	return style.Render("Testing Project") + "\n" +
		spinnerStyle.Render(c.spinnerFrame) + " " + style.Render("Running tests...") +
		output + "\n\n" +
		style.Render("Press q to quit")
}

func (c *Component) renderInfo() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ffaa")).
		Bold(true)
	return style.Render(c.infoMsg)
}

func (c *Component) renderError() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ff0000")).
		Bold(true)
	return style.Render(c.errorMsg)
}

type DownloadProgressMsg struct{ Progress float64 }
type DownloadCompleteMsg struct{ Variant *api.Project }
type DownloadErrorMsg struct{ Error string }
type TestCompleteMsg struct {
	Variant *api.Project
	Result  interface{} // Will be the test result from testrunner
}
type TestErrorMsg struct{ Error string }
type BackMsg struct{}
type QuitMsg struct{}

// Spinner frames and message type
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type spinnerMsg struct{ frame string }
