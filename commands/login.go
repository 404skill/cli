package commands

import (
    "bufio"
    "fmt"
    "os"
    "strings"
    "context"
    "time"

    "github.com/joho/godotenv"
    "log"
    "404skill-cli/supabase"
    "404skill-cli/auth"
    "404skill-cli/config"
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

    accessToken, err := authProvider.SignIn(context.Background(), c.Username, c.Password); 
    if err != nil {
        fmt.Println("Invalid credentials")
        return nil
    } else {
        fmt.Println("Access Token:", accessToken)
    }


    cfg := config.Config{
        Username:    c.Username,
        Password:    c.Password,
        AccessToken: accessToken,
        LastUpdated: time.Now(),
    }

    err = config.WriteConfig(cfg)
    if err != nil {
        return err
    }

    fmt.Println("Login successful, configuration saved.")
    return nil
} 