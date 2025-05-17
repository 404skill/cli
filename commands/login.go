package commands

import (
    "bufio"
    "fmt"
    "os"
    "strings"
    "context"

    "github.com/joho/godotenv"
    "log"
    "404skill-cli/supabase"
    "404skill-cli/auth"
)

func init() {
    err := godotenv.Load()
    if err != nil {
        log.Fatal("Error loading .env file")
    }
}

type LoginCmd struct {
    Username string `short:"u" long:"username" description:"Username for login"`
    Password string `short:"p" long:"password" description:"Password for login"`
}

func (c *LoginCmd) Execute(args []string) error {
    reader := bufio.NewReader(os.Stdin)

    if c.Username == "" {
        fmt.Print("Enter username: ")
        username, _ := reader.ReadString('\n')
        c.Username = strings.TrimSpace(username)
    }

    if c.Password == "" {
        fmt.Print("Enter password: ")
        password, _ := reader.ReadString('\n')
        c.Password = strings.TrimSpace(password)
    }

    client, err := supabase.NewSupabaseClient()
    if err != nil {
        fmt.Println(err)
        return err
    }

    authProvider := auth.NewSupabaseAuth(client)
    if accessToken, err := authProvider.SignIn(context.Background(), c.Username, c.Password); err != nil {
        fmt.Println("Invalid credentials")
        return nil
    } else {
        fmt.Println("Access Token:", accessToken)
    }

    // call to our backend should go here to generate / get the api key.
    apiKey := "mock-api-key"

    // Save API key to ~/.404skill/config.yml
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return err
    }

    // Ensure the directory exists
    err = os.MkdirAll(fmt.Sprintf("%s/.404skill", homeDir), os.ModePerm)
    if err != nil {
        return err
    }

    configPath := fmt.Sprintf("%s/.404skill/config.yml", homeDir)

    file, err := os.OpenFile(configPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
    if err != nil {
        return err
    }
    defer file.Close()

    _, err = file.WriteString(fmt.Sprintf("api_key: %s\n", apiKey))
    if err != nil {
        return err
    }

    fmt.Println("Login successful, API key saved.")
    return nil
} 