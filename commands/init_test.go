package commands

import (
    "context"
    "errors"
    "testing"
    "404skill-cli/api"
)

// mockClient is a mock implementation of the API client
type mockClient struct {
    template *api.ProjectTemplate
    err      error
}

func (m *mockClient) ListProjects(ctx context.Context) ([]api.Project, error) {
    return nil, nil
}

func (m *mockClient) InitProject(ctx context.Context, projectIdentifier string) (*api.ProjectTemplate, error) {
    return m.template, m.err
}

// mockDownloader is a mock implementation of the template downloader
type mockDownloader struct {
    extractedDir string
    err          error
}

func (m *mockDownloader) DownloadAndExtract(url, targetDir string) (string, error) {
    return m.extractedDir, m.err
}

func TestInitCmd_Execute(t *testing.T) {
    tests := []struct {
        name          string
        projectID     string
        mockTemplate  *api.ProjectTemplate
        mockAPIErr    error
        mockExtractDir string
        mockExtractErr error
        wantErr       bool
    }{
        {
            name:      "successful initialization",
            projectID: "test-project",
            mockTemplate: &api.ProjectTemplate{
                DownloadURL: "http://example.com/template.zip",
                ProjectName: "test-project",
            },
            mockAPIErr:     nil,
            mockExtractDir: "./test-project",
            mockExtractErr: nil,
            wantErr:       false,
        },
        {
            name:          "missing project ID",
            projectID:     "",
            mockTemplate:  nil,
            mockAPIErr:    nil,
            mockExtractDir: "",
            mockExtractErr: nil,
            wantErr:       true,
        },
        {
            name:          "API error",
            projectID:     "test-project",
            mockTemplate:  nil,
            mockAPIErr:    errors.New("API error"),
            mockExtractDir: "",
            mockExtractErr: nil,
            wantErr:       true,
        },
        {
            name:      "extraction error",
            projectID: "test-project",
            mockTemplate: &api.ProjectTemplate{
                DownloadURL: "http://example.com/template.zip",
                ProjectName: "test-project",
            },
            mockAPIErr:     nil,
            mockExtractDir: "",
            mockExtractErr: errors.New("extraction error"),
            wantErr:       true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cmd := &InitCmd{
                client: &mockClient{
                    template: tt.mockTemplate,
                    err:     tt.mockAPIErr,
                },
                downloader: &mockDownloader{
                    extractedDir: tt.mockExtractDir,
                    err:         tt.mockExtractErr,
                },
                ProjectID: tt.projectID,
            }

            err := cmd.Execute([]string{})

            if (err != nil) != tt.wantErr {
                t.Errorf("InitCmd.Execute() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
} 