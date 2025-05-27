package tui

import (
	"context"
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
	"github.com/charmbracelet/lipgloss"
	btable "github.com/evertras/bubble-table/table"
)

var successStyle = headerStyle.Copy().Foreground(lipgloss.Color("2"))

// spinnerFrames holds the frames for our animated spinner.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// TestComponent handles project testing functionality.
type TestComponent struct {
	table               btable.Model
	projects            []api.Project
	client              api.ClientInterface
	testing             bool
	errorMsg            string
	fileManager         FileManager
	configManager       ConfigManager
	help                help.Model
	spinnerFrame        string
	outputBuffer        []string
	currentTestProject  *api.Project
	currentTestCmdState *testCmdState
	showingTestResults  bool
	testResultsSummary  string
	testResultsList     []string
}

// testResultMsg contains the parsed test results.
type testResultMsg struct {
	result *testreport.ParseResult
	err    error
}

// outputLineMsg is sent when a new line of output is available,
// or when the command completes (with done=true).
type outputLineMsg struct {
	line string
	done bool
	err  error
}

// testCmdState holds the state for a running test command.
type testCmdState struct {
	cmd   *exec.Cmd
	lines chan outputLineMsg
}

// nextMsgCmd returns a Cmd that will pull exactly one message off of lines.
func (s *testCmdState) nextMsgCmd() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-s.lines
		if !ok {
			return nil
		}
		return msg
	}
}

// NewTestComponent creates a new TestComponent.
func NewTestComponent(fileManager FileManager, configManager ConfigManager, client api.ClientInterface) *TestComponent {
	rows := []btable.Row{}
	table := btable.New(bubbleTableColumns).WithRows(rows)

	return &TestComponent{
		table:         table,
		fileManager:   fileManager,
		configManager: configManager,
		help:          help.New(),
		spinnerFrame:  spinnerFrames[0],
		client:        client,
	}
}

// Update handles incoming messages and updates state.
func (t *TestComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if t.showingTestResults {
			// Any key returns to project list
			t.showingTestResults = false
			t.testResultsSummary = ""
			t.testResultsList = nil
			return t, nil
		}
		if msg.String() == "enter" {
			selected := t.table.HighlightedRow()
			if selected.Data != nil {
				if id, ok := selected.Data["id"].(string); ok {
					for _, p := range t.projects {
						if p.ID == id {
							t.testing = true
							t.errorMsg = ""
							t.currentTestProject = &p
							return t, tea.Batch(
								t.runTests(p),
								t.spinnerTick(),
							)
						}
					}
				}
			}
		}

	case []api.Project:
		t.projects = msg
		rows := []btable.Row{}
		cfg, err := config.ReadConfig()
		if err != nil {
			t.errorMsg = fmt.Sprintf("Failed to read config: %v", err)
			return t, nil
		}
		for _, p := range msg {
			if cfg.DownloadedProjects != nil && cfg.DownloadedProjects[p.ID] {
				rows = append(rows, btable.NewRow(map[string]interface{}{
					"id":     p.ID,
					"name":   p.Name,
					"lang":   p.Language,
					"diff":   p.Difficulty,
					"dur":    fmt.Sprintf("%d min", p.EstimatedDurationInMinutes),
					"status": "✓ Downloaded",
				}))
			}
		}
		t.table = btable.New(bubbleTableColumns).WithRows(rows).Focused(true)

	case testResultMsg:
		if msg.err != nil {
			t.errorMsg = msg.err.Error()
			return t, nil
		}
		// Build summary
		t.showingTestResults = true
		t.testing = false
		testCount := msg.result.Suite.Tests
		passedCount := len(msg.result.PassedTests)
		failedCount := len(msg.result.FailedTests)
		testTime := msg.result.Suite.Time
		t.testResultsSummary = fmt.Sprintf(
			"%s\n\nTotal: %d   Passed: %d   Failed: %d   Time: %.2fs",
			headerStyle.Render("Test Results: "+msg.result.Suite.Name),
			testCount, passedCount, failedCount, testTime,
		)
		// Build list of all tests with status and time
		var list []string
		for _, tr := range msg.result.Suite.Results {
			status := ""
			if tr.Passed {
				status = successStyle.Render("[PASS]")
			} else {
				status = errorStyle.Render("[FAIL]")
			}
			list = append(list, fmt.Sprintf("%s  %s  (%.2fs)", status, tr.Name, tr.Time))
		}
		t.testResultsList = list

		err := t.client.BulkUpdateProfileTests(context.Background(), msg.result.FailedTests, msg.result.PassedTests, t.currentTestProject.ID)
		if err != nil {
			t.testResultsSummary += "\n\n[API update failed: " + err.Error() + "]"
		} else {
			t.testResultsSummary += "\n\n[API update successful!]"
		}
		return t, nil

	case spinnerMsg:
		t.spinnerFrame = msg.frame
		if t.testing {
			return t, t.spinnerTick()
		}
		return t, nil

	case outputLineMsg:
		if msg.err != nil && !msg.done {
			t.testing = false
			t.errorMsg = msg.err.Error()
			return t, nil
		}
		if msg.line != "" {
			t.outputBuffer = append(t.outputBuffer, msg.line)
			return t, t.currentTestCmdState.nextMsgCmd()
		}
		if msg.done {
			t.testing = false
			if msg.err != nil {
				t.errorMsg = msg.err.Error()
			}
			return t, t.parseTestReportAfterRun()
		}
		return t, nil
	}

	t.table, cmd = t.table.Update(msg)
	return t, cmd
}

// View renders the TestComponent UI.
func (t *TestComponent) View() string {
	if t.showingTestResults {
		var b strings.Builder
		b.WriteString(t.testResultsSummary)
		b.WriteString("\n\n")
		for _, line := range t.testResultsList {
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("\nPress any key to return to the project list.")
		return b.String()
	}
	if t.testing {
		out := strings.Join(t.outputBuffer, "\n")
		return fmt.Sprintf("%s\n\nRunning tests...\n%s\n%s\n\nPress q to quit",
			headerStyle.Render("Testing Project"),
			spinnerStyle.Render(t.spinnerFrame),
			out)
	}

	helpView := helpStyle.Render(t.help.View(keys) + "  [esc/b] back")
	view := fmt.Sprintf("%s\n%s", t.table.View(), helpView)
	if t.errorMsg != "" {
		view = fmt.Sprintf("%s\n\n%s", view, errorStyle.Render(t.errorMsg))
	}
	return view
}

// runTests starts docker-compose, streams its output (splitting on \r or \n),
// and returns the first message immediately.
func (t *TestComponent) runTests(project api.Project) tea.Cmd {
	return func() tea.Msg {
		home, err := os.UserHomeDir()
		if err != nil {
			return outputLineMsg{err: fmt.Errorf("home dir: %w", err), done: true}
		}

		repo := strings.ToLower(strings.ReplaceAll(project.Name, " ", "_"))
		base := filepath.Join(home, "404skill_projects")
		ents, err := os.ReadDir(base)
		if err != nil {
			return outputLineMsg{err: fmt.Errorf("read projects dir: %w", err), done: true}
		}

		var projectDir string
		for _, e := range ents {
			if e.IsDir() && strings.HasPrefix(e.Name(), repo) {
				projectDir = filepath.Join(base, e.Name())
				break
			}
		}
		if projectDir == "" {
			return outputLineMsg{
				err:  fmt.Errorf("project dir not found for '%s'", repo),
				done: true,
			}
		}

		cmd := exec.Command("docker", "compose", "up", "--build", "--abort-on-container-exit")
		cmd.Dir = projectDir
		cmd.Run()

		return outputLineMsg{done: true}
	}
}

// spinnerMsg is used to update which frame of the spinner to show.
type spinnerMsg struct{ frame string }

// spinnerTick returns a Cmd that waits 100ms then sends the next spinner frame.
func (t *TestComponent) spinnerTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
		idx := 0
		for i, f := range spinnerFrames {
			if f == t.spinnerFrame {
				idx = i
				break
			}
		}
		return spinnerMsg{spinnerFrames[(idx+1)%len(spinnerFrames)]}
	})
}

// parseTestReportAfterRun looks for the XML report and parses it.
func (t *TestComponent) parseTestReportAfterRun() tea.Cmd {
	return func() tea.Msg {
		home, err := os.UserHomeDir()
		if err != nil {
			return errMsg{err: fmt.Errorf("home dir: %w", err)}
		}
		if t.currentTestProject == nil {
			return errMsg{err: fmt.Errorf("no project context")}
		}

		repo := strings.ToLower(strings.ReplaceAll(t.currentTestProject.Name, " ", "_"))
		base := filepath.Join(home, "404skill_projects")
		ents, err := os.ReadDir(base)
		if err != nil {
			return errMsg{err: fmt.Errorf("read projects dir: %w", err)}
		}

		var projectDir string
		for _, e := range ents {
			if e.IsDir() && strings.HasPrefix(e.Name(), repo) {
				projectDir = filepath.Join(base, e.Name())
				break
			}
		}
		if projectDir == "" {
			return errMsg{err: fmt.Errorf("project dir missing")}
		}

		reports := filepath.Join(base, ".tests", fmt.Sprintf("%s_%s", repo, t.currentTestProject.Language), "test-reports")
		ents, err = os.ReadDir(reports)
		if err != nil {
			return errMsg{err: fmt.Errorf("read reports dir: %w", err)}
		}

		var xmlPath string
		for _, e := range ents {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".xml") {
				xmlPath = filepath.Join(reports, e.Name())
				break
			}
		}
		if xmlPath == "" {
			return errMsg{err: fmt.Errorf("no .xml in %s", reports)}
		}

		parser := testreport.NewParser()
		res, err := parser.ParseFile(xmlPath)
		if err != nil {
			return errMsg{err: fmt.Errorf("parse report: %w", err)}
		}
		return testResultMsg{result: res}
	}
}
