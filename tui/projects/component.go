package projects

import (
	"404skill-cli/api"
	"404skill-cli/config"
	"404skill-cli/filesystem"
	"404skill-cli/tui/components/table"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Component handles project listing, selection, and downloaded project management
type Component struct {
	// Dependencies
	client        api.ClientInterface
	configManager *config.ConfigManager
	fileManager   *filesystem.Manager

	// UI components
	table *table.Component

	// State
	projects []api.Project
	loading  bool
	errorMsg string
	ready    bool
}

// New creates a new projects component with dependency injection
func New(client api.ClientInterface, configManager *config.ConfigManager, fileManager *filesystem.Manager) *Component {
	comp := &Component{
		client:        client,
		configManager: configManager,
		fileManager:   fileManager,
		loading:       false,
	}

	// Create table component with this component as the status provider
	comp.table = table.New(comp)

	return comp
}

// GetProjectStatus implements table.ProjectStatusProvider interface
func (c *Component) GetProjectStatus(projectID string) string {
	if c.configManager.IsProjectDownloaded(projectID) {
		return "✓ Downloaded"
	}
	return ""
}

// SetLoading sets the loading state
func (c *Component) SetLoading(loading bool) {
	c.loading = loading
}

// SetProjects updates the projects list
func (c *Component) SetProjects(projects []api.Project) {
	c.projects = projects
	c.table.SetProjects(projects)
	c.table.SetFocused(true)
	c.loading = false
}

// SetError sets an error message
func (c *Component) SetError(err string) {
	c.errorMsg = err
	c.loading = false
}

// GetSelectedProject returns the currently highlighted project
func (c *Component) GetSelectedProject() *api.Project {
	return c.table.GetHighlightedProject()
}

// Update handles component updates
func (c *Component) Update(msg tea.Msg) (*Component, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			selectedProject := c.table.GetHighlightedProject()
			if selectedProject != nil {
				// Check if project is already downloaded
				if c.configManager.IsProjectDownloaded(selectedProject.ID) {
					// Handle downloaded project
					return c, c.handleDownloadedProject(selectedProject)
				}

				// Project not downloaded, emit selection message
				return c, func() tea.Msg {
					return ProjectSelectedMsg{Project: selectedProject}
				}
			}
		}
	case []api.Project:
		c.SetProjects(msg)
		return c, nil
	case ProjectsErrorMsg:
		c.SetError(msg.Error)
		return c, nil
	}

	// Update table component
	c.table, cmd = c.table.Update(msg)

	return c, cmd
}

// handleDownloadedProject handles when a user selects an already downloaded project
func (c *Component) handleDownloadedProject(project *api.Project) tea.Cmd {
	return func() tea.Msg {
		// Try to open the project directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return ProjectsErrorMsg{Error: "Project already downloaded but couldn't determine home directory."}
		}

		// Format project name for directory
		repoName := strings.ToLower(strings.ReplaceAll(project.Name, " ", "_"))
		projectsDir := filepath.Join(homeDir, "404skill_projects")

		// Try to find the project directory
		entries, err := os.ReadDir(projectsDir)
		if err != nil {
			return ProjectsErrorMsg{Error: "Project already downloaded but couldn't access projects directory."}
		}

		var projectDir string
		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(entry.Name(), repoName) {
				projectDir = filepath.Join(projectsDir, entry.Name())
				break
			}
		}

		if projectDir == "" {
			// Project directory not found, offer redownload
			return ProjectRedownloadNeededMsg{Project: project}
		}

		// Try to open the directory
		if err := c.fileManager.OpenFileExplorer(projectDir); err != nil {
			return ProjectsErrorMsg{Error: fmt.Sprintf("Project was downloaded but couldn't open directory: %v", err)}
		}

		return ProjectOpenedMsg{Message: "Project already downloaded. Opening project directory..."}
	}
}

// View renders the component
func (c *Component) View() string {
	if c.loading {
		return c.renderLoading()
	}

	view := c.table.View()

	if c.errorMsg != "" {
		view += "\n\n" + c.renderError()
	}

	return view
}

// renderLoading renders the loading state
func (c *Component) renderLoading() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ffaa")).
		Bold(true).
		Underline(true).
		Padding(0, 1)
	return style.Render("\nLoading projects...")
}

// renderError renders error messages
func (c *Component) renderError() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ff0000")).
		Bold(true)
	return style.Render(c.errorMsg)
}

// UpdateProjectStatus refreshes the project status in the table
func (c *Component) UpdateProjectStatus() {
	c.table.UpdateProjectStatus()
}
