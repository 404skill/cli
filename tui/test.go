package tui

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"404skill-cli/api"
	"404skill-cli/config"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	btable "github.com/evertras/bubble-table/table"
)

// spinnerFrames holds the frames for our animated spinner.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// TestComponent handles project testing functionality.
type TestComponent struct {
	table         btable.Model
	projects      []api.Project
	selected      int
	testing       bool
	errorMsg      string
	fileManager   FileManager
	configManager ConfigManager
	help          help.Model
	spinnerFrame  string
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
					// Find the selected project
					for _, proj := range t.projects {
						if proj.ID == projectID {
							t.testing = true
							t.errorMsg = ""
							// Kick off both the test runner and spinner ticker
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

	case testCompleteMsg:
		t.testing = false
		return t, nil

	case errMsg:
		t.errorMsg = msg.err.Error()
		t.testing = false
		return t, nil

	case spinnerMsg:
		// Update spinner frame
		t.spinnerFrame = msg.frame
		// If still testing, schedule next tick
		if t.testing {
			return t, t.spinnerTick()
		}
		return t, nil
	}

	// Let the table component handle other messages
	t.table, cmd = t.table.Update(msg)
	return t, cmd
}

// View renders the TestComponent UI.
func (t *TestComponent) View() string {
	if t.testing {
		return fmt.Sprintf("%s\n\nRunning tests...\n%s\n\nPress q to quit",
			headerStyle.Render("Testing Project"),
			spinnerStyle.Render(t.spinnerFrame))
	}

	helpView := helpStyle.Render(t.help.View(keys) + "  [esc/b] back")
	view := fmt.Sprintf("%s\n%s", t.table.View(), helpView)
	if t.errorMsg != "" {
		view = fmt.Sprintf("%s\n\n%s", view, errorStyle.Render(t.errorMsg))
	}
	return view
}

// runTests returns a Cmd that runs docker-compose and emits a message when done.
func (t *TestComponent) runTests(project api.Project) tea.Cmd {
	return func() tea.Msg {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to get home directory: %w", err)}
		}

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

		cmd := exec.Command("docker", "compose", "up", "-d", "--build", "--abort-on-container-exit")
		cmd.Dir = projectDir

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			out := stderr.String()
			if out != "" {
				return errMsg{err: fmt.Errorf("command failed: %s", out)}
			}
			return errMsg{err: err}
		}

		return testCompleteMsg{}
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
