package commands

import (
	"404skill-cli/api"
	"context"
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
)

// Project represents the metadata of a project
type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListCmd represents the list command
type ListCmd struct {
	client api.ClientInterface
}

// NewListCmd creates a new instance of ListCmd
func NewListCmd(client api.ClientInterface) *ListCmd {
	return &ListCmd{
		client: client,
	}
}

// Execute implements the Command interface
func (c *ListCmd) Execute(args []string) error {
	ctx := context.Background()
	projects, err := c.client.ListProjects(ctx)
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	// Print the projects in a table format
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"ID", "Name"})
	table.Bulk(projects)
	table.Render()
	return nil
}
