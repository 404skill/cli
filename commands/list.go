package commands

import (
    "fmt"
    "404skill-cli/config"
    "net/http"
    "io/ioutil"
    "os"
    "encoding/json"
    "github.com/olekukonko/tablewriter"
)

// Project represents the metadata of a project
type Project struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

// ListCmd handles the `list` command
type ListCmd struct{}

func (c *ListCmd) Execute(args []string) error {
    token, err := config.GetToken()
    if err != nil {
        fmt.Println(err)
        return err
    }

    baseURL := config.GetBaseURL()

    req, err := http.NewRequest("GET", fmt.Sprintf("%s/hello", baseURL), nil)
    if err != nil {
        fmt.Println("Error creating request:", err)
        return err
    }

    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

    // Send the request
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        fmt.Println("Error making request:", err)
        return err
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        fmt.Println("Error reading response:", err)
        return err
    }

    fmt.Println("Response from /hello:", string(body))

    var projects []Project
    if err := json.Unmarshal(body, &projects); err != nil {
        fmt.Println("Error deserializing response:", err)
        return err
    }

    table := tablewriter.NewWriter(os.Stdout)
    table.Header([]string{"ID", "Name"})
    table.Bulk(projects)
    table.Render()

    return nil
} 