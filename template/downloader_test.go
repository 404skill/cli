package template

import (
	"archive/zip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloader_DownloadAndExtract(t *testing.T) {
	// Create a test zip file
	zipContent := []byte("test content")
	zipFile := createTestZip(t, map[string][]byte{
		"test.txt": zipContent,
	})

	tests := []struct {
		name           string
		mockStatusCode int
		mockResponse   []byte
		wantErr        bool
	}{
		{
			name:           "successful download and extract",
			mockStatusCode: http.StatusOK,
			mockResponse:   zipFile,
			wantErr:        false,
		},
		{
			name:           "download error",
			mockStatusCode: http.StatusInternalServerError,
			mockResponse:   nil,
			wantErr:        true,
		},
		{
			name:           "invalid zip",
			mockStatusCode: http.StatusOK,
			mockResponse:   []byte("invalid zip content"),
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.mockStatusCode)
				if tt.mockResponse != nil {
					w.Write(tt.mockResponse)
				}
			}))
			defer server.Close()

			// Create temporary directory for extraction
			tempDir, err := os.MkdirTemp("", "test-extract-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Test download and extract
			downloader := NewDownloader()
			extractedDir, err := downloader.DownloadAndExtract(server.URL, tempDir)

			if (err != nil) != tt.wantErr {
				t.Errorf("DownloadAndExtract() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify extracted content
				content, err := os.ReadFile(filepath.Join(extractedDir, "test.txt"))
				if err != nil {
					t.Errorf("Failed to read extracted file: %v", err)
					return
				}

				if string(content) != string(zipContent) {
					t.Errorf("DownloadAndExtract() content = %v, want %v", string(content), string(zipContent))
				}
			}
		})
	}
}

// createTestZip creates a test zip file with the given files
func createTestZip(t *testing.T, files map[string][]byte) []byte {
	t.Helper()

	// Create a temporary file for the zip
	tmpFile, err := os.CreateTemp("", "test-*.zip")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Create zip writer
	writer := zip.NewWriter(tmpFile)
	defer writer.Close()

	// Add files to zip
	for name, content := range files {
		f, err := writer.Create(name)
		if err != nil {
			t.Fatalf("Failed to create zip entry: %v", err)
		}
		if _, err := f.Write(content); err != nil {
			t.Fatalf("Failed to write zip content: %v", err)
		}
	}

	// Close the writer to flush the zip
	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close zip writer: %v", err)
	}

	// Read the zip file
	zipContent, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read zip file: %v", err)
	}

	return zipContent
}
