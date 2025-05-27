package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockTokenProvider is a mock implementation of the token provider
type mockTokenProvider struct {
	token string
	err   error
}

func (m *mockTokenProvider) GetToken() (string, error) {
	return m.token, m.err
}

func TestClient_ListProjects(t *testing.T) {
	// Test cases
	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		wantProjects   []Project
	}{
		{
			name: "successful list",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Method != http.MethodGet {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				if r.Header.Get("Authorization") != "Bearer test-token" {
					t.Errorf("expected Authorization header 'Bearer test-token', got %s", r.Header.Get("Authorization"))
				}

				// Return success response
				projects := []Project{
					{ID: "1", Name: "Project One"},
					{ID: "2", Name: "Project Two"},
				}
				json.NewEncoder(w).Encode(projects)
			},
			wantErr: false,
			wantProjects: []Project{
				{ID: "1", Name: "Project One"},
				{ID: "2", Name: "Project Two"},
			},
		},
		{
			name: "api error",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "Internal server error",
				})
			},
			wantErr:      true,
			wantProjects: nil,
		},
		{
			name: "invalid response",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("invalid json"))
			},
			wantErr:      true,
			wantProjects: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			// Create client with mock token provider
			tokenProvider := &mockTokenProvider{
				token: "test-token",
			}
			if tt.name == "missing token" {
				tokenProvider.token = ""
			}
			client := &Client{
				httpClient:    &http.Client{},
				baseURL:       server.URL,
				tokenProvider: tokenProvider,
			}

			// Make request
			projects, err := client.ListProjects(context.Background())

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.ListProjects() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check projects if no error
			if !tt.wantErr {
				if len(projects) != len(tt.wantProjects) {
					t.Errorf("got %d projects, want %d", len(projects), len(tt.wantProjects))
					return
				}
				for i, p := range projects {
					if p != tt.wantProjects[i] {
						t.Errorf("project[%d] = %v, want %v", i, p, tt.wantProjects[i])
					}
				}
			}
		})
	}
}

func TestClient_InitProject(t *testing.T) {
	// Test cases
	tests := []struct {
		name           string
		projectID      string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		wantTemplate   *ProjectTemplate
	}{
		{
			name:      "successful init",
			projectID: "test-project",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Method != http.MethodGet {
					t.Errorf("expected POST request, got %s", r.Method)
				}
				if r.Header.Get("Authorization") != "Bearer test-token" {
					t.Errorf("expected Authorization header 'Bearer test-token', got %s", r.Header.Get("Authorization"))
				}

				// Return success response
				template := &ProjectTemplate{
					DownloadURL: "https://example.com/template.zip",
					ProjectName: "test-project",
				}
				json.NewEncoder(w).Encode(template)
			},
			wantErr: false,
			wantTemplate: &ProjectTemplate{
				DownloadURL: "https://example.com/template.zip",
				ProjectName: "test-project",
			},
		},
		{
			name:      "api error",
			projectID: "test-project",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "Internal server error",
				})
			},
			wantErr:      true,
			wantTemplate: nil,
		},
		{
			name:      "invalid response",
			projectID: "test-project",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("invalid json"))
			},
			wantErr:      true,
			wantTemplate: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			// Create client with mock token provider
			tokenProvider := &mockTokenProvider{
				token: "test-token",
			}
			if tt.name == "missing token" {
				tokenProvider.token = ""
			}
			client := &Client{
				httpClient:    &http.Client{},
				baseURL:       server.URL,
				tokenProvider: tokenProvider,
			}

			// Make request
			template, err := client.InitProject(context.Background(), tt.projectID)

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.InitProject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check template if no error
			if !tt.wantErr {
				if template == nil {
					t.Error("expected template, got nil")
					return
				}
				if template.DownloadURL != tt.wantTemplate.DownloadURL {
					t.Errorf("got DownloadURL %s, want %s", template.DownloadURL, tt.wantTemplate.DownloadURL)
				}
				if template.ProjectName != tt.wantTemplate.ProjectName {
					t.Errorf("got ProjectName %s, want %s", template.ProjectName, tt.wantTemplate.ProjectName)
				}
			}
		})
	}
}
