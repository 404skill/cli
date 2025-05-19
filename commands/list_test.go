package commands

import (
    "context"
    "errors"
    "testing"
    "404skill-cli/api"
)

// MockClient is a mock implementation of the API client
type MockClient struct {
    projects []api.Project
    err      error
}

func (m *MockClient) ListProjects(ctx context.Context) ([]api.Project, error) {
    return m.projects, m.err
}

func (m *MockClient) InitProject(ctx context.Context, projectIdentifier string) (*api.ProjectTemplate, error) {
    return nil, nil
}

func TestListCmd_Execute(t *testing.T) {
    // Test cases
    tests := []struct {
        name    string
        mock    api.ClientInterface
        wantErr bool
    }{
        {
            name: "successful list",
            mock: &MockClient{
                projects: []api.Project{
                    {ID: "1", Name: "Project One"},
                    {ID: "2", Name: "Project Two"},
                },
                err: nil,
            },
            wantErr: false,
        },
        {
            name: "api error",
            mock: &MockClient{
                projects: nil,
                err:     errors.New("api error"),
            },
            wantErr: true,
        },
        {
            name: "empty list",
            mock: &MockClient{
                projects: []api.Project{},
                err:     nil,
            },
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cmd := NewListCmd(tt.mock)

            err := cmd.Execute([]string{})

            if (err != nil) != tt.wantErr {
                t.Errorf("ListCmd.Execute() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
} 