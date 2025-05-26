package tui

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"404skill-cli/api"
	"404skill-cli/config"
	"404skill-cli/testreport"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	btable "github.com/evertras/bubble-table/table"
)

// spinnerFrames holds the frames for our animated spinner.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// TestComponent handles project testing functionality.
type TestComponent struct {
	table              btable.Model
	projects           []api.Project
	selected           int
	testing            bool
	errorMsg           string
	fileManager        FileManager
	configManager      ConfigManager
	help               help.Model
	spinnerFrame       string
	outputBuffer       []string
	currentTestProject *api.Project
}

// NewTestComponent creates a new TestComponent.
func NewTestComponent(fileManager FileManager, configManager ConfigManager) *TestComponent {
	rows := []btable.Row{}
	table := btable.New(bubbleTableColumns).WithRows(rows)

	return &TestComponent{
		table:         table,
		fileManager:   fileManager,
		configManager: configManager,
		help:          help.New(),
		spinnerFrame:  spinnerFrames[0],
	}
}

// Update handles incoming messages and updates state.
func (t *TestComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			selectedRow := t.table.HighlightedRow()
			if selectedRow.Data != nil {
				if projectID, ok := selectedRow.Data["id"].(string); ok {
					for _, proj := range t.projects {
						if proj.ID == projectID {
							t.testing = true
							t.errorMsg = ""
							t.currentTestProject = &proj

							return t, tea.Batch(
								t.runTests(proj),
								t.spinnerTick(),
							)
						}
					}
				}
			}
		}
	case []api.Project:
		// Populate table with downloaded projects
		t.projects = msg
		rows := []btable.Row{}

		cfg, err := config.ReadConfig()
		if err != nil {
			t.errorMsg = fmt.Sprintf("Failed to read config: %v", err)
			return t, nil
		}

		for _, proj := range msg {
			if cfg.DownloadedProjects != nil && cfg.DownloadedProjects[proj.ID] {
				rows = append(rows, btable.NewRow(map[string]interface{}{
					"id":     proj.ID,
					"name":   proj.Name,
					"lang":   proj.Language,
					"diff":   proj.Difficulty,
					"dur":    fmt.Sprintf("%d min", proj.EstimatedDurationInMinutes),
					"status": "✓ Downloaded",
				}))
			}
		}

		t.table = btable.New(bubbleTableColumns).
			WithRows(rows).
			Focused(true)

	case testResultMsg:
		t.testing = false
		if msg.err != nil {
			t.errorMsg = msg.err.Error()
			return t, nil
		}

		// Format test results
		var result strings.Builder
		result.WriteString(fmt.Sprintf("Test Suite: %s\n", msg.result.Suite.Name))
		result.WriteString(fmt.Sprintf("Total Tests: %d\n", msg.result.Suite.Tests))
		result.WriteString(fmt.Sprintf("Passed: %d\n", len(msg.result.PassedTests)))
		result.WriteString(fmt.Sprintf("Failed: %d\n", len(msg.result.FailedTests)))
		result.WriteString(fmt.Sprintf("Time: %.2fs\n\n", msg.result.Suite.Time))

		if len(msg.result.FailedTests) > 0 {
			result.WriteString("Failed Tests:\n")
			for _, test := range msg.result.FailedTests {
				result.WriteString(fmt.Sprintf("- %s\n", test))
			}
		}

		t.errorMsg = result.String()
		return t, nil

	case spinnerMsg:
		// Update spinner frame
		t.spinnerFrame = msg.frame
		// If still testing, schedule next tick
		if t.testing {
			return t, t.spinnerTick()
		}
		return t, nil

	case outputLineMsg:
		if msg.err != nil {
			t.testing = false
			t.errorMsg = msg.err.Error()
			return t, nil
		}
		if msg.line != "" {
			t.outputBuffer = append(t.outputBuffer, msg.line)
		}
		if msg.done {
			// After command finishes, look for test reports as before
			return t, t.parseTestReportAfterRun()
		}
		// Continue reading output
		return t, nil
	}

	// Let the table component handle other messages
	t.table, cmd = t.table.Update(msg)
	return t, cmd
}

// View renders the TestComponent UI.
func (t *TestComponent) View() string {
	if t.testing {
		output := strings.Join(t.outputBuffer, "\n")
		return fmt.Sprintf("%s\n\nRunning tests...\n%s\n%s\n\nPress q to quit",
			headerStyle.Render("Testing Project"),
			spinnerStyle.Render(t.spinnerFrame),
			output)
	}

	helpView := helpStyle.Render(t.help.View(keys) + "  [esc/b] back")
	view := fmt.Sprintf("%s\n%s", t.table.View(), helpView)
	if t.errorMsg != "" {
		view = fmt.Sprintf("%s\n\n%s", view, errorStyle.Render(t.errorMsg))
	}
	return view
}

// testResultMsg contains the parsed test results
type testResultMsg struct {
	result *testreport.ParseResult
	err    error
}

// Msg for new output line
// outputLineMsg is sent when a new line of output is available
// or when the command completes (with done=true)
type outputLineMsg struct {
	line string
	done bool
	err  error
}

// runTests returns a Cmd that runs docker-compose and streams output lines.
func (t *TestComponent) runTests(project api.Project) tea.Cmd {
	return func() tea.Msg {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return outputLineMsg{err: fmt.Errorf("failed to get home directory: %w", err), done: true}
		}

		repoName := strings.ToLower(strings.ReplaceAll(project.Name, " ", "_"))
		projectsDir := filepath.Join(homeDir, "404skill_projects")

		entries, err := os.ReadDir(projectsDir)
		if err != nil {
			return outputLineMsg{err: fmt.Errorf("failed to read projects directory: %w", err), done: true}
		}

		var projectDir string
		dirsFound := []string{} // collect all dir names for debug
		for _, entry := range entries {
			if entry.IsDir() {
				dirsFound = append(dirsFound, entry.Name())
				if strings.HasPrefix(entry.Name(), repoName) {
					projectDir = filepath.Join(projectsDir, entry.Name())
					break
				}
			}
		}

		if projectDir == "" {
			dirsList := strings.Join(dirsFound, ", ")
			return outputLineMsg{err: fmt.Errorf("project directory not found. Looking for prefix: '%s' in: [%s]", repoName, dirsList), done: true}
		}

		cmd := exec.Command("docker", "compose", "up", "-d", "--build", "--abort-on-container-exit")
		cmd.Dir = projectDir

		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			return outputLineMsg{err: fmt.Errorf("failed to get stdout pipe: %w", err), done: true}
		}
		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			return outputLineMsg{err: fmt.Errorf("failed to get stderr pipe: %w", err), done: true}
		}

		if err := cmd.Start(); err != nil {
			return outputLineMsg{err: fmt.Errorf("failed to start command: %w", err), done: true}
		}

		// Stream output lines from both stdout and stderr
		outputChan := make(chan outputLineMsg)
		go func() {
			defer close(outputChan)
			stdoutScanner := bufio.NewScanner(stdoutPipe)
			stderrScanner := bufio.NewScanner(stderrPipe)
			for stdoutScanner.Scan() {
				outputChan <- outputLineMsg{line: stdoutScanner.Text()}
			}
			for stderrScanner.Scan() {
				outputChan <- outputLineMsg{line: stderrScanner.Text()}
			}
			if err := cmd.Wait(); err != nil {
				outputChan <- outputLineMsg{err: fmt.Errorf("command failed: %w", err), done: true}
				return
			}
			outputChan <- outputLineMsg{done: true}
		}()

		// Return a tea.Cmd that reads from outputChan and sends messages
		return func() tea.Msg {
			for msg := range outputChan {
				return msg
			}
			return nil
		}
	}
}

// spinnerTick returns a Cmd that waits 100ms then sends the next spinner frame.
func (t *TestComponent) spinnerTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(_ time.Time) tea.Msg {
		// find current spinner index
		idx := 0
		for i, f := range spinnerFrames {
			if f == t.spinnerFrame {
				idx = i
				break
			}
		}
		next := spinnerFrames[(idx+1)%len(spinnerFrames)]
		return spinnerMsg{frame: next}
	})
}

// spinnerMsg is used to update which frame of the spinner to show.
type spinnerMsg struct {
	frame string
}

// testCompleteMsg signals that the docker-compose run has finished.
type testCompleteMsg struct{}

// parseTestReportAfterRun returns a Cmd that looks for and parses the test report after the command
func (t *TestComponent) parseTestReportAfterRun() tea.Cmd {
	return func() tea.Msg {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to get home directory: %w", err)}
		}

		if t.currentTestProject == nil {
			return errMsg{err: fmt.Errorf("no project context for test report lookup")}
		}
		project := *t.currentTestProject
		repoName := strings.ToLower(strings.ReplaceAll(project.Name, " ", "_"))
		projectsDir := filepath.Join(homeDir, "404skill_projects")
		entries, err := os.ReadDir(projectsDir)
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to read projects directory: %w", err)}
		}
		var projectDir string
		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(entry.Name(), repoName) {
				projectDir = filepath.Join(projectsDir, entry.Name())
				break
			}
		}
		if projectDir == "" {
			return errMsg{err: fmt.Errorf("project directory not found")}
		}
		reportsDir := filepath.Join(projectDir, "test-reports")
		entries, err = os.ReadDir(reportsDir)
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to read reports directory: %w", err)}
		}
		var reportPath string
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".xml") {
				reportPath = filepath.Join(reportsDir, entry.Name())
				break
			}
		}
		if reportPath == "" {
			return errMsg{err: fmt.Errorf("no test report found in %s", reportsDir)}
		}
		parser := testreport.NewParser()
		result, err := parser.ParseFile(reportPath)
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to parse test report: %w", err)}
		}
		return testResultMsg{result: result}
	}
}
