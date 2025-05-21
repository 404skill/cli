package main

import (
    "os"

    "github.com/jessevdk/go-flags"
    "404skill-cli/commands"
    "404skill-cli/api"
    "404skill-cli/config"
)

func main() {
    // Create API client with config token provider
    tokenProvider := config.NewConfigTokenProvider()
    client := api.NewClient(tokenProvider)

    parser := flags.NewParser(nil, flags.Default)
    parser.AddCommand("login", "Login to 404skill", "Prompts for user credentials and saves API key.", &commands.LoginCmd{})
    parser.AddCommand("init", "Initialize a project", "Downloads project files.", commands.NewInitCmd(client))
    parser.AddCommand("test", "Run tests", "Executes tests and updates task status.", &commands.TestCmd{})
    parser.AddCommand("list", "List projects", "Shows project IDs and slugs.", commands.NewListCmd(client))

    if _, err := parser.Parse(); err != nil {
        os.Exit(1)
    }
} 