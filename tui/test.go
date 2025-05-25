package tui

import (
	"404skill-cli/api"
	"404skill-cli/config"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	btable "github.com/evertras/bubble-table/table"
)

// TestComponent handles project testing functionality
type TestComponent struct {
	table         btable.Model
	projects      []api.Project
	selected      int
	testing       bool
	errorMsg      string
	fileManager   FileManager
	configManager ConfigManager
	help          help.Model
}

// NewTestComponent creates a new test component
func NewTestComponent(fileManager FileManager, configManager ConfigManager) *TestComponent {
	rows := []btable.Row{}
	table := btable.New(bubbleTableColumns).WithRows(rows)

	return &TestComponent{
		table:         table,
		fileManager:   fileManager,
		configManager: configManager,
		help:          help.New(),
	}
}

// Update handles messages for the test component
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
							return t, t.runTests(proj)
						}
					}
				}
			}
		}
	case []api.Project:
		// Filter only downloaded projects
		t.projects = msg
		rows := []btable.Row{}

		// Get downloaded projects from config
		cfg, err := config.ReadConfig()
		if err != nil {
			t.errorMsg = fmt.Sprintf("Failed to read config: %v", err)
			return t, nil
		}

		for _, proj := range msg {
			isDownloaded := cfg.DownloadedProjects != nil && cfg.DownloadedProjects[proj.ID]
			fmt.Printf("Project %s (ID: %s) downloaded: %v\n", proj.Name, proj.ID, isDownloaded)

			if isDownloaded {
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
	}

	t.table, cmd = t.table.Update(msg)
	return t, cmd
}

// View renders the test component
func (t *TestComponent) View() string {
	if t.testing {
		return fmt.Sprintf("%s\n\nRunning tests...\n%s\n\nPress q to quit",
			headerStyle.Render("Testing Project"),
			spinnerStyle.Render("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"))
	}

	helpView := helpStyle.Render(t.help.View(keys) + "  [esc/b] back")
	view := fmt.Sprintf("%s\n%s", t.table.View(), helpView)
	if t.errorMsg != "" {
		view = fmt.Sprintf("%s\n\n%s", view, errorStyle.Render(t.errorMsg))
	}
	return view
}

// runTests executes the docker-compose test command for the selected project
func (t *TestComponent) runTests(project api.Project) tea.Cmd {
	return func() tea.Msg {
		// Get home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to get home directory: %w", err)}
		}

		// Format project name for directory
		repoName := strings.ToLower(strings.ReplaceAll(project.Name, " ", "_"))
		projectsDir := filepath.Join(homeDir, "404skill_projects")

		// Find the project directory
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

		// Run docker-compose command
		cmd := exec.Command("docker", "compose", "up", "-d", "--build", "--abort-on-container-exit")
		cmd.Dir = projectDir

		// Capture output but don't display it
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			// If there's an error, include the command output in the error message
			errOutput := stderr.String()
			if errOutput != "" {
				return errMsg{err: fmt.Errorf("command failed: %s", errOutput)}
			}
			return errMsg{err: fmt.Errorf("command failed: %w", err)}
		}

		return testCompleteMsg{}
	}
}

// testCompleteMsg is sent when the test execution completes
type testCompleteMsg struct{}
