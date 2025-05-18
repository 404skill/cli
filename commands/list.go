package commands

import (
    "fmt"
    "404skill-cli/config"
    "net/http"
    "io/ioutil"
)

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

    token, err := config.GetToken()
    if err != nil {
        fmt.Println(err)
        return err
    }

    fmt.Println(token)

    fmt.Println("Available projects:")
    for _, project := range projects {
        fmt.Printf("ID: %s, Slug: %s\n", project.ID, project.Slug)
    }

    // Create a new HTTP request
    req, err := http.NewRequest("GET", "http://localhost:8080/hello", nil)
    if err != nil {
        fmt.Println("Error creating request:", err)
        return err
    }

    // Set the Authorization header
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

    // Send the request
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        fmt.Println("Error making request:", err)
        return err
    }
    defer resp.Body.Close()

    // Read and print the response body
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        fmt.Println("Error reading response:", err)
        return err
    }

    fmt.Println("Response from /hello:", string(body))

    return nil
} 