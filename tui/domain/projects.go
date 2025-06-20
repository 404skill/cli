package domain

import (
	"404skill-cli/api"
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	btable "github.com/evertras/bubble-table/table"
)

// ProjectService handles project-related operations
type ProjectService struct {
	client api.ClientInterface
}

// NewProjectService creates a new project service
func NewProjectService(client api.ClientInterface) *ProjectService {
	return &ProjectService{
		client: client,
	}
}

// FetchProjects fetches projects from the API
func (s *ProjectService) FetchProjects() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		projects, err := s.client.ListProjects(ctx)
		if err != nil {
			return ProjectsErrorMsg{Error: err}
		}
		return ProjectsLoadedMsg{Projects: projects}
	}
}

// ProjectUtils provides utility functions for project operations
type ProjectUtils struct{}

// NewProjectUtils creates a new project utilities instance
func NewProjectUtils() *ProjectUtils {
	return &ProjectUtils{}
}

// ExtractUniqueNames extracts unique project names from a list of projects
func (u *ProjectUtils) ExtractUniqueNames(projects []api.Project) []string {
	seen := make(map[string]struct{})
	var names []string

	for _, p := range projects {
		if _, exists := seen[p.Name]; !exists {
			seen[p.Name] = struct{}{}
			names = append(names, p.Name)
		}
	}

	return names
}

// FilterByName filters projects by name
func (u *ProjectUtils) FilterByName(projects []api.Project, name string) []api.Project {
	var filtered []api.Project
	for _, p := range projects {
		if p.Name == name {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// FormatVariantsTable formats project variants into a readable table string
func (u *ProjectUtils) FormatVariantsTable(variants []api.Project) string {
	if len(variants) == 0 {
		return "No variants found."
	}

	// Define headers
	headers := []interface{}{
		"Technologies", "Difficulty", "Language", "Description",
		"Repo URL", "Type", "Estimated", "Access Tier",
	}

	rowFormat := "%-20s %-10s %-10s %-15s %-30s %-10s %-9s %-10s\n"

	// Build header row
	result := fmt.Sprintf(rowFormat, headers...)
	result += strings.Repeat("-", 120) + "\n"

	// Build data rows
	for _, variant := range variants {
		result += fmt.Sprintf(rowFormat,
			variant.Technologies,
			variant.Difficulty,
			variant.Language,
			variant.Description,
			variant.RepoUrl,
			variant.Type,
			fmt.Sprintf("%d", variant.EstimatedDurationInMinutes),
			variant.AccessTier,
		)
	}

	return result
}

// CreateTableColumns creates standard table columns for project display
func (u *ProjectUtils) CreateTableColumns() []btable.Column {
	return []btable.Column{
		btable.NewColumn("name", "Name", 32),
		btable.NewColumn("lang", "Language", 15),
		btable.NewColumn("diff", "Difficulty", 15),
		btable.NewColumn("dur", "Duration", 15),
		btable.NewColumn("status", "Status", 15),
	}
}

// Messages for project domain events
type (
	// ProjectsLoadedMsg is sent when projects are successfully loaded
	ProjectsLoadedMsg struct {
		Projects []api.Project
	}

	// ProjectsErrorMsg is sent when there's an error loading projects
	ProjectsErrorMsg struct {
		Error error
	}

	// ProjectSelectedMsg is sent when a project is selected
	ProjectSelectedMsg struct {
		Project *api.Project
	}

	// ProjectNameSelectedMsg is sent when a project name is selected (for variant selection)
	ProjectNameSelectedMsg struct {
		ProjectName string
		Variants    []api.Project
	}
)
