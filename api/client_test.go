package api

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"
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
        mockResponse   []Project
        mockStatusCode int
        mockToken     string
        wantErr       bool
    }{
        {
            name: "successful response",
            mockResponse: []Project{
                {ID: "1", Name: "Project One"},
                {ID: "2", Name: "Project Two"},
            },
            mockStatusCode: http.StatusOK,
            mockToken:     "test-token",
            wantErr:       false,
        },
        {
            name:           "server error",
            mockResponse:   nil,
            mockStatusCode: http.StatusInternalServerError,
            mockToken:     "test-token",
            wantErr:       true,
        },
        {
            name:           "invalid response",
            mockResponse:   nil,
            mockStatusCode: http.StatusOK,
            mockToken:     "test-token",
            wantErr:       true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Create a test server
            server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                // Verify request headers
                authHeader := r.Header.Get("Authorization")
                if authHeader != "Bearer "+tt.mockToken {
                    t.Errorf("expected Authorization header 'Bearer %s', got '%s'", tt.mockToken, authHeader)
                }

                // Set response status code
                w.WriteHeader(tt.mockStatusCode)

                // Write mock response
                if tt.mockResponse != nil {
                    json.NewEncoder(w).Encode(tt.mockResponse)
                } else {
                    w.Write([]byte("invalid json"))
                }
            }))
            defer server.Close()

            // Create client with test server URL and mock token provider
            client := &Client{
                httpClient: &http.Client{
                    Timeout: 5 * time.Second,
                },
                baseURL: server.URL,
                tokenProvider: &mockTokenProvider{
                    token: tt.mockToken,
                },
            }

            // Execute test
            projects, err := client.ListProjects(context.Background())

            // Check error
            if (err != nil) != tt.wantErr {
                t.Errorf("ListProjects() error = %v, wantErr %v", err, tt.wantErr)
                return
            }

            // Check response
            if !tt.wantErr {
                if len(projects) != len(tt.mockResponse) {
                    t.Errorf("ListProjects() got %d projects, want %d", len(projects), len(tt.mockResponse))
                }
                for i, p := range projects {
                    if p.ID != tt.mockResponse[i].ID || p.Name != tt.mockResponse[i].Name {
                        t.Errorf("ListProjects() project[%d] = %v, want %v", i, p, tt.mockResponse[i])
                    }
                }
            }
        })
    }
} 