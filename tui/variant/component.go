package variant

import (
	"404skill-cli/api"
	"404skill-cli/config"
	"404skill-cli/downloader"
	"404skill-cli/filesystem"
	"404skill-cli/testrunner"
	"404skill-cli/tracing"
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
	verboseMode      bool
	highLevelStatus  string
	filteredMessages []string
	tracer           *tracing.TUIIntegration
}

func New(variants []api.Project, downloader downloader.Downloader, configManager *config.ConfigManager, fileManager *filesystem.Manager) *Component {
	return NewWithMode(variants, downloader, nil, configManager, fileManager, DownloadMode)
}

func NewForTesting(variants []api.Project, testRunner testrunner.TestRunner, configManager *config.ConfigManager, fileManager *filesystem.Manager) *Component {
	return NewWithMode(variants, nil, testRunner, configManager, fileManager, TestMode)
}

func NewWithMode(variants []api.Project, downloader downloader.Downloader, testRunner testrunner.TestRunner, configManager *config.ConfigManager, fileManager *filesystem.Manager, mode Mode) *Component {
	// Get tracing integration from global manager
	var tuiTracer *tracing.TUIIntegration
	if manager := tracing.GetGlobalManager(); manager != nil {
		tuiTracer = tracing.NewTUIIntegration(manager)
	}

	// Create center alignment style for all columns
	centerStyle := lipgloss.NewStyle().Align(lipgloss.Center)

	columns := []btable.Column{
		btable.NewColumn("desc", "Description", 32).WithStyle(centerStyle),
		btable.NewColumn("tech", "Technologies", 24).WithStyle(centerStyle),
		btable.NewColumn("diff", "Difficulty", 12).WithStyle(centerStyle),
		btable.NewColumn("downloaded", "Downloaded", 12).WithStyle(centerStyle),
	}
	var rows []btable.Row
	for _, v := range variants {
		downloadedStatus := "✗"
		if configManager != nil && configManager.IsProjectDownloaded(v.ID) {
			downloadedStatus = "✓"
		}

		rows = append(rows, btable.NewRow(map[string]interface{}{
			"desc":       v.Description,
			"tech":       v.Technologies,
			"diff":       v.Difficulty,
			"downloaded": downloadedStatus,
		}))
	}
	table := btable.New(columns).WithRows(rows).Focused(true)

	component := &Component{
		variants:      variants,
		configManager: configManager,
		fileManager:   fileManager,
		downloader:    downloader,
		testRunner:    testRunner,
		table:         table,
		selectedIdx:   0,
		mode:          mode,
		tracer:        tuiTracer,
	}

	// Track component initialization
	if tuiTracer != nil {
		modeStr := "download"
		if mode == TestMode {
			modeStr = "test"
		}
		_ = tuiTracer.TrackProjectOperation("variant_component_init", fmt.Sprintf("%s_mode", modeStr))
	}

	return component
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
			if c.tracer != nil {
				_ = c.tracer.TrackProjectOperation("download_complete", msg.Variant.Name)
			}
			c.downloading = false
			c.selectedVariant = msg.Variant
			c.refreshTable()
			return c, nil
		case DownloadErrorMsg:
			if c.tracer != nil {
				_ = c.tracer.TrackError(fmt.Errorf("%s", msg.Error), "variant", "download")
			}
			c.downloading = false
			c.errorMsg = msg.Error
			return c, nil
		}
		return c, c.progressTicker()
	}

	if c.testing {
		switch msg := msg.(type) {
		case TestCompleteMsg:
			if c.tracer != nil {
				_ = c.tracer.TrackProjectOperation("test_complete", msg.Variant.Name)
			}
			c.testing = false
			c.selectedVariant = msg.Variant
			return c, nil
		case TestErrorMsg:
			if c.tracer != nil {
				_ = c.tracer.TrackError(fmt.Errorf("%s", msg.Error), "variant", "test")
			}
			c.testing = false
			c.errorMsg = msg.Error
			return c, nil
		case spinnerMsg:
			c.spinnerFrame = msg.frame
			return c, c.spinnerTick()
		case tea.KeyMsg:
			switch msg.String() {
			case "v":
				if c.tracer != nil {
					_ = c.tracer.TrackKeyMsg(msg, "variant_testing_verbose_toggle")
				}
				c.verboseMode = !c.verboseMode
				return c, nil
			case "q", "ctrl+c":
				if c.tracer != nil {
					_ = c.tracer.TrackKeyMsg(msg, "variant_testing_quit")
				}
				return c, func() tea.Msg { return QuitMsg{} }
			}
		}
		return c, c.spinnerTick()
	}

	c.table, _ = c.table.Update(msg)

	if m, ok := msg.(tea.KeyMsg); ok {
		switch m.String() {
		case "up", "k":
			if c.tracer != nil {
				_ = c.tracer.TrackKeyMsg(m, "variant_navigation")
			}
			if c.selectedIdx > 0 {
				c.selectedIdx--
			}
		case "down", "j":
			if c.tracer != nil {
				_ = c.tracer.TrackKeyMsg(m, "variant_navigation")
			}
			if c.selectedIdx < len(c.variants)-1 {
				c.selectedIdx++
			}
		case "enter":
			if c.tracer != nil {
				_ = c.tracer.TrackKeyMsg(m, "variant_selection")
			}
			if c.selectedIdx >= 0 && c.selectedIdx < len(c.variants) {
				variant := c.variants[c.selectedIdx]
				if c.mode == DownloadMode {
					return c.handleDownloadAction(&variant)
				} else {
					return c.handleTestAction(&variant)
				}
			}
		case "esc", "b":
			if c.tracer != nil {
				_ = c.tracer.TrackKeyMsg(m, "variant_back_navigation")
			}
			return c, func() tea.Msg { return BackMsg{} }
		case "q", "ctrl+c":
			if c.tracer != nil {
				_ = c.tracer.TrackKeyMsg(m, "variant_quit")
			}
			return c, func() tea.Msg { return QuitMsg{} }
		}
	}
	return c, nil
}

func (c *Component) handleDownloadAction(variant *api.Project) (*Component, tea.Cmd) {
	// Track download action initiation
	if c.tracer != nil {
		_ = c.tracer.TrackMenuNavigation("variant_table", "download_action", variant.Name)
	}

	if c.configManager != nil && c.configManager.IsProjectDownloaded(variant.ID) {
		if c.tracer != nil {
			_ = c.tracer.TrackProjectOperation("project_already_downloaded", variant.Name)
		}

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
						if c.tracer != nil {
							fileTracker := c.tracer.TrackFileOperation("open_project_directory", projectDir)
							_ = fileTracker.Complete()
						}
						_ = c.fileManager.OpenFileExplorer(projectDir)
					}
				}
			}
		}
		c.infoMsg = "Project already downloaded. Opening project directory..."
		return c, nil
	}

	// Track new download initiation
	if c.tracer != nil {
		_ = c.tracer.TrackProjectOperation("download_start", variant.Name)
	}

	return c, c.downloadWithProgress(variant)
}

func (c *Component) handleTestAction(variant *api.Project) (*Component, tea.Cmd) {
	// Track test action initiation
	if c.tracer != nil {
		_ = c.tracer.TrackMenuNavigation("variant_table", "test_action", variant.Name)
	}

	// Check if project is downloaded
	if c.configManager == nil || !c.configManager.IsProjectDownloaded(variant.ID) {
		if c.tracer != nil {
			_ = c.tracer.TrackError(fmt.Errorf("project not downloaded"), "variant", "test_prerequisite_check")
		}
		c.errorMsg = "Project must be downloaded before testing. Please download it first."
		return c, nil
	}

	// Track test initiation
	if c.tracer != nil {
		_ = c.tracer.TrackProjectOperation("test_start", variant.Name)
	}

	// Start testing with spinner
	c.testing = true
	c.verboseMode = false // Start in simple mode
	c.currentOperation = "Initializing tests..."
	c.highLevelStatus = "Preparing to run tests..."
	c.spinnerFrame = spinnerFrames[0]
	c.outputBuffer = []string{}     // Clear previous output
	c.filteredMessages = []string{} // Clear previous filtered messages
	c.errorMsg = ""                 // Clear previous errors
	c.infoMsg = ""                  // Clear previous info
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
		// Track download operation
		var downloadTracker *tracing.TimedOperationTracker
		if c.tracer != nil {
			downloadTracker = c.tracer.TrackProjectOperation("download_execution", variant.Name)
			downloadTracker.AddMetadata("project_id", variant.ID)
			downloadTracker.AddMetadata("language", variant.Language)
			downloadTracker.AddMetadata("difficulty", variant.Difficulty)
		}

		ctx := context.Background()
		progressCallback := func(progress float64) {
			atomic.StoreUint64(&c.atomicProgress, uint64(progress*100))
		}
		c.SetDownloading(true)
		c.currentOperation = "Cloning project..."
		err := c.downloader.DownloadProject(ctx, variant, variant.Language, progressCallback)

		if err != nil {
			if downloadTracker != nil {
				_ = downloadTracker.CompleteWithError(err)
			}
			return DownloadErrorMsg{Error: err.Error()}
		}

		if downloadTracker != nil {
			_ = downloadTracker.Complete()
		}

		return DownloadCompleteMsg{Variant: variant}
	}
}

func (c *Component) startTest(variant *api.Project) tea.Cmd {
	return func() tea.Msg {
		// Track test operation
		var testTracker *tracing.TimedOperationTracker
		if c.tracer != nil {
			testTracker = c.tracer.TrackProjectOperation("test_execution", variant.Name)
			testTracker.AddMetadata("project_id", variant.ID)
			testTracker.AddMetadata("language", variant.Language)
			testTracker.AddMetadata("difficulty", variant.Difficulty)
		}

		// Convert api.Project to testrunner.Project
		testProject := testrunner.Project{
			ID:       variant.ID,
			Name:     variant.Name,
			Language: variant.Language,
		}

		// Progress callback for test runner - update component state with filtering
		progressCallback := func(message string) {
			c.processProgressMessage(message)
		}

		// Run tests
		result, err := c.testRunner.RunTests(testProject, progressCallback)
		if err != nil {
			if testTracker != nil {
				_ = testTracker.CompleteWithError(err)
			}
			return TestErrorMsg{Error: err.Error()}
		}

		if testTracker != nil {
			_ = testTracker.Complete()
		}

		return TestCompleteMsg{Variant: variant, Result: result}
	}
}

// Helper methods for message processing
func (c *Component) extractHighLevelStatus(message string) string {
	// Extract high-level status from common patterns
	if strings.Contains(message, "> Task :test") {
		if strings.Contains(message, "UP-TO-DATE") {
			return "Tests are up-to-date"
		} else if strings.Contains(message, "NO-SOURCE") {
			return "No test sources found"
		} else {
			return "Running tests..."
		}
	}
	if strings.Contains(message, "> Task :build") {
		return "Building project..."
	}
	if strings.Contains(message, "> Task :compile") {
		return "Compiling sources..."
	}
	if strings.Contains(message, "BUILD SUCCESSFUL") {
		return "✅ Build completed successfully"
	}
	if strings.Contains(message, "BUILD FAILED") {
		return "❌ Build failed"
	}
	if strings.Contains(message, "Starting docker-compose") {
		return "Starting Docker containers..."
	}
	if strings.Contains(message, "Docker-compose finished") {
		return "Docker containers finished"
	}
	return ""
}

func (c *Component) shouldShowInBasicMode(message string) bool {
	// Hide Docker build noise
	dockerNoisePatterns := []string{
		"#", "CACHED", "DONE ", "exporting layers", "writing image",
		"transferring context", "transferring dockerfile", "FromAsCasing",
		"internal] load", "auth]", "resolving provenance", "pull token",
		"Container .* Recreat", "Attaching to",
	}

	for _, pattern := range dockerNoisePatterns {
		if strings.Contains(message, pattern) {
			return false
		}
	}

	// Show meaningful content
	meaningfulPatterns := []string{
		"> Task :", "BUILD ", "actionable tasks:", "exited with code",
		"Starting", "Stopping", "Stopped", "ERROR", "FAILED", "SUCCESS",
	}

	for _, pattern := range meaningfulPatterns {
		if strings.Contains(message, pattern) {
			return true
		}
	}

	return false
}

func (c *Component) cleanMessage(message string) string {
	// Remove prefixes like "OUT: " and "ERR: "
	cleaned := strings.TrimSpace(message)
	if strings.HasPrefix(cleaned, "OUT: ") {
		cleaned = strings.TrimPrefix(cleaned, "OUT: ")
	}
	if strings.HasPrefix(cleaned, "ERR: ") {
		cleaned = strings.TrimPrefix(cleaned, "ERR: ")
	}
	return cleaned
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

	modeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Italic(true)

	// Header with spinner
	header := style.Render("Testing Project") + "\n" +
		spinnerStyle.Render(c.spinnerFrame) + " " + style.Render(c.highLevelStatus)

	// Mode indicator and instructions
	var modeInfo string
	var output string

	if c.verboseMode {
		// Verbose mode - show all output
		modeInfo = modeStyle.Render("(Verbose Mode - showing all output)")
		if len(c.outputBuffer) > 0 {
			// Show last 10 lines of full output
			start := 0
			if len(c.outputBuffer) > 10 {
				start = len(c.outputBuffer) - 10
			}
			outputLines := c.outputBuffer[start:]
			output = "\n" + outputStyle.Render(strings.Join(outputLines, "\n"))
		}
	} else {
		// Simple mode - show filtered meaningful content
		modeInfo = modeStyle.Render("(Simple Mode - showing key updates)")
		if len(c.filteredMessages) > 0 {
			output = "\n" + outputStyle.Render(strings.Join(c.filteredMessages, "\n"))
		}
	}

	// Footer with controls
	controlsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666"))

	controls := controlsStyle.Render("Press [v] to toggle verbose mode • [q] to quit")

	return header + "\n" + modeInfo + output + "\n\n" + controls
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

// processProgressMessage handles incoming progress messages and updates component state
func (c *Component) processProgressMessage(message string) {
	// Always store full message for verbose mode
	c.outputBuffer = append(c.outputBuffer, message)
	// Keep only last 20 messages to prevent memory issues
	if len(c.outputBuffer) > 20 {
		c.outputBuffer = c.outputBuffer[len(c.outputBuffer)-20:]
	}

	// Update high-level status for simple mode
	if status := c.extractHighLevelStatus(message); status != "" {
		c.highLevelStatus = status
	}

	// Store filtered message for basic mode
	if c.shouldShowInBasicMode(message) {
		c.filteredMessages = append(c.filteredMessages, c.cleanMessage(message))
		// Keep only last 8 filtered messages
		if len(c.filteredMessages) > 8 {
			c.filteredMessages = c.filteredMessages[len(c.filteredMessages)-8:]
		}
	}

	c.currentOperation = message
}

// Getter methods
func (c *Component) IsTesting() bool {
	return c.testing
}

func (c *Component) IsDownloading() bool {
	return c.downloading
}

func (c *Component) refreshTable() {
	// Create center alignment style for all columns
	centerStyle := lipgloss.NewStyle().Align(lipgloss.Center)

	columns := []btable.Column{
		btable.NewColumn("desc", "Description", 32).WithStyle(centerStyle),
		btable.NewColumn("tech", "Technologies", 24).WithStyle(centerStyle),
		btable.NewColumn("diff", "Difficulty", 12).WithStyle(centerStyle),
		btable.NewColumn("downloaded", "Downloaded", 12).WithStyle(centerStyle),
	}
	var rows []btable.Row
	for _, v := range c.variants {
		downloadedStatus := "✗"
		if c.configManager != nil && c.configManager.IsProjectDownloaded(v.ID) {
			downloadedStatus = "✓"
		}

		rows = append(rows, btable.NewRow(map[string]interface{}{
			"desc":       v.Description,
			"tech":       v.Technologies,
			"diff":       v.Difficulty,
			"downloaded": downloadedStatus,
		}))
	}
	c.table = btable.New(columns).WithRows(rows).Focused(true)
}
