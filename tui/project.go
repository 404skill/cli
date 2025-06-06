package tui

import (
	"404skill-cli/api"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	btable "github.com/evertras/bubble-table/table"
)

// ProjectComponent handles the project list and selection
type ProjectComponent struct {
	table         btable.Model
	viewport      viewport.Model
	help          help.Model
	client        api.ClientInterface
	configManager ConfigManager
	projects      []api.Project
	selected      int
	loading       bool
	errorMsg      string
	selectedInfo  string
	ready         bool
}

// NewProjectComponent creates a new project component
func NewProjectComponent(client api.ClientInterface, configManager ConfigManager) *ProjectComponent {
	rows := []btable.Row{}
	table := btable.New(bubbleTableColumns).WithRows(rows)

	return &ProjectComponent{
		table:         table,
		help:          help.New(),
		client:        client,
		configManager: configManager,
	}
}

// Init initializes the project component
func (p *ProjectComponent) Init() tea.Cmd {
	return nil
}

// Update handles messages for the project component
func (p *ProjectComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			selectedRow := p.table.HighlightedRow()
			if selectedRow.Data != nil {
				// Check if project is already downloaded
				if projectID, ok := selectedRow.Data["id"].(string); ok && p.configManager.IsProjectDownloaded(projectID) {
					// Try to open the project directory
					homeDir, err := os.UserHomeDir()
					if err != nil {
						p.errorMsg = "Project already downloaded but couldn't determine home directory."
						return p, nil
					}

					// Find the project in our list to get its name
					var projectName string
					for _, proj := range p.projects {
						if proj.ID == projectID {
							projectName = proj.Name
							break
						}
					}

					if projectName == "" {
						p.errorMsg = "Project already downloaded but couldn't find project details."
						return p, nil
					}

					// Format project name for directory
					repoName := strings.ToLower(strings.ReplaceAll(projectName, " ", "_"))
					projectsDir := filepath.Join(homeDir, "404skill_projects")

					// Try to find the project directory
					entries, err := os.ReadDir(projectsDir)
					if err != nil {
						p.errorMsg = "Project already downloaded but couldn't access projects directory."
						return p, nil
					}

					var projectDir string
					for _, entry := range entries {
						if entry.IsDir() && strings.HasPrefix(entry.Name(), repoName) {
							projectDir = filepath.Join(projectsDir, entry.Name())
							break
						}
					}

					if projectDir == "" {
						p.errorMsg = "Project was downloaded but directory not found. It might have been moved or deleted."
						return p, nil
					}

					// Try to open the directory
					if err := openFileExplorer(projectDir); err != nil {
						p.errorMsg = fmt.Sprintf("Project was downloaded but couldn't open directory: %v", err)
						return p, nil
					}

					p.errorMsg = "Project already downloaded. Opening project directory..."
					return p, nil
				}

				// Find the selected project
				for _, proj := range p.projects {
					if proj.ID == selectedRow.Data["id"] {
						return p, tea.Batch(
							func() tea.Msg { return projectSelectedMsg{project: proj} },
						)
					}
				}
			}
		}
	case tea.WindowSizeMsg:
		if !p.ready {
			p.ready = true
			p.viewport = viewport.New(msg.Width, msg.Height-7)
			p.viewport.Style = baseStyle
		}
	case []api.Project:
		p.projects = msg
		rows := []btable.Row{}

		// Get downloaded projects from config
		downloadedProjects := p.configManager.GetDownloadedProjects()

		for _, proj := range msg {
			status := ""
			if downloadedProjects[proj.ID] {
				status = "✓ Downloaded"
			}
			rows = append(rows, btable.NewRow(map[string]interface{}{
				"id":     proj.ID,
				"name":   proj.Name,
				"lang":   proj.Language,
				"diff":   proj.Difficulty,
				"dur":    fmt.Sprintf("%d min", proj.EstimatedDurationInMinutes),
				"status": status,
			}))
		}
		p.table = btable.New(bubbleTableColumns).
			WithRows(rows).
			Focused(true)

		p.loading = false
	case errMsg:
		p.errorMsg = msg.err.Error()
		p.loading = false
	}

	p.table, cmd = p.table.Update(msg)
	return p, cmd
}

// View renders the project component
func (p *ProjectComponent) View() string {
	if p.loading {
		return headerStyle.Render("\nLoading projects...")
	}

	helpView := helpStyle.Render(p.help.View(keys) + "  [esc/b] back")
	info := ""
	if p.selectedInfo != "" {
		info = "\n" + p.selectedInfo
	}
	view := fmt.Sprintf("%s\n%s%s", p.table.View(), helpView, info)
	if p.errorMsg != "" {
		view = fmt.Sprintf("%s\n\n%s", view, errorStyle.Render(p.errorMsg))
	}
	return view
}

// projectSelectedMsg is sent when a project is selected
type projectSelectedMsg struct {
	project api.Project
}
