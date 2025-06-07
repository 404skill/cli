package table

import (
	"fmt"

	"404skill-cli/api"

	tea "github.com/charmbracelet/bubbletea"
	btable "github.com/evertras/bubble-table/table"
)

// ProjectStatusProvider defines how to get the status for a project
type ProjectStatusProvider interface {
	GetProjectStatus(projectID string) string
}

// Component represents a reusable project table
type Component struct {
	table          btable.Model
	projects       []api.Project
	statusProvider ProjectStatusProvider
	focused        bool
}

// New creates a new table component with default styling
func New(statusProvider ProjectStatusProvider) *Component {
	// Define consistent column structure
	columns := []btable.Column{
		btable.NewColumn("name", "Name", 32),
		btable.NewColumn("lang", "Language", 15),
		btable.NewColumn("diff", "Difficulty", 15),
		btable.NewColumn("dur", "Duration", 15),
		btable.NewColumn("status", "Status", 15),
	}

	table := btable.New(columns)

	return &Component{
		table:          table,
		statusProvider: statusProvider,
		focused:        false,
	}
}

// SetProjects updates the table with new project data
func (c *Component) SetProjects(projects []api.Project) {
	c.projects = projects
	c.refreshTable()
}

// SetFocused sets whether the table should be focused
func (c *Component) SetFocused(focused bool) {
	c.focused = focused
	if focused {
		c.table = c.table.Focused(true)
	} else {
		c.table = c.table.Focused(false)
	}
}

// GetHighlightedProject returns the currently highlighted project
func (c *Component) GetHighlightedProject() *api.Project {
	selectedRow := c.table.HighlightedRow()
	if selectedRow.Data == nil {
		return nil
	}

	if id, ok := selectedRow.Data["id"].(string); ok {
		for _, p := range c.projects {
			if p.ID == id {
				return &p
			}
		}
	}
	return nil
}

// Update handles Bubble Tea messages
func (c *Component) Update(msg tea.Msg) (*Component, tea.Cmd) {
	var cmd tea.Cmd
	c.table, cmd = c.table.Update(msg)
	return c, cmd
}

// View renders the table
func (c *Component) View() string {
	return c.table.View()
}

// refreshTable rebuilds the table rows from current project data
func (c *Component) refreshTable() {
	var rows []btable.Row

	for _, p := range c.projects {
		status := ""
		if c.statusProvider != nil {
			status = c.statusProvider.GetProjectStatus(p.ID)
		}

		rows = append(rows, btable.NewRow(map[string]interface{}{
			"id":     p.ID,
			"name":   p.Name,
			"lang":   p.Language,
			"diff":   p.Difficulty,
			"dur":    fmt.Sprintf("%d min", p.EstimatedDurationInMinutes),
			"status": status,
		}))
	}

	c.table = c.table.WithRows(rows)
	if c.focused {
		c.table = c.table.Focused(true)
	}
}

// UpdateProjectStatus refreshes the table to reflect current project statuses
func (c *Component) UpdateProjectStatus() {
	c.refreshTable()
}
