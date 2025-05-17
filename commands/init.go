package commands

import "fmt"

// InitCmd handles the `init` command
type InitCmd struct {
    Project string `short:"p" long:"project" description:"Project slug or ID" required:"true"`
}

func (c *InitCmd) Execute(args []string) error {
    fmt.Printf("init called: downloading project '%s' into current directory\n", c.Project)
    return nil
} 