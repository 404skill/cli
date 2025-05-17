package main

import (
    "os"

    "github.com/jessevdk/go-flags"
    "404skill-cli/commands"
)

func main() {
    parser := flags.NewParser(nil, flags.Default)
    parser.AddCommand("login", "Login to 404skill", "Prompts for user credentials and saves API key.", &commands.LoginCmd{})
    parser.AddCommand("init", "Initialize a project", "Downloads project files.", &commands.InitCmd{})
    parser.AddCommand("test", "Run tests", "Executes tests and updates task status.", &commands.TestCmd{})
    parser.AddCommand("list", "List projects", "Shows project IDs and slugs.", &commands.ListCmd{})

    if _, err := parser.Parse(); err != nil {
        os.Exit(1)
    }
} 