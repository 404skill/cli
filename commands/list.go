package commands

import "fmt"

// ListCmd handles the `list` command
type ListCmd struct{}

func (c *ListCmd) Execute(args []string) error {
    projects := []struct {
        ID   string
        Slug string
    }{
        {ID: "1", Slug: "project-one"},
        {ID: "2", Slug: "project-two"},
        {ID: "3", Slug: "project-three"},
    }

    fmt.Println("Available projects:")
    for _, project := range projects {
        fmt.Printf("ID: %s, Slug: %s\n", project.ID, project.Slug)
    }
    return nil
} 