package template

import (
    "archive/zip"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
)

// DownloaderInterface defines the interface for template downloaders
type DownloaderInterface interface {
    DownloadAndExtract(url, targetDir string) (string, error)
}

// Downloader handles template downloads and extraction
type Downloader struct {
    httpClient *http.Client
}

// NewDownloader creates a new template downloader
func NewDownloader() *Downloader {
    return &Downloader{
        httpClient: &http.Client{},
    }
}

// DownloadAndExtract downloads a template and extracts it to the specified directory
func (d *Downloader) DownloadAndExtract(url, targetDir string) (string, error) {
    // Create a temporary file for the zip
    tmpFile, err := os.CreateTemp("", "template-*.zip")
    if err != nil {
        return "", fmt.Errorf("failed to create temp file: %w", err)
    }
    defer os.Remove(tmpFile.Name())
    defer tmpFile.Close()

    // Download the template
    resp, err := d.httpClient.Get(url)
    if err != nil {
        return "", fmt.Errorf("failed to download template: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("failed to download template: status code %d", resp.StatusCode)
    }

    // Save the zip file
    if _, err := io.Copy(tmpFile, resp.Body); err != nil {
        return "", fmt.Errorf("failed to save template: %w", err)
    }

    // Create target directory if it doesn't exist
    if err := os.MkdirAll(targetDir, 0755); err != nil {
        return "", fmt.Errorf("failed to create target directory: %w", err)
    }

    // Extract the zip file
    reader, err := zip.OpenReader(tmpFile.Name())
    if err != nil {
        return "", fmt.Errorf("failed to open zip file: %w", err)
    }
    defer reader.Close()

    // Extract each file
    for _, file := range reader.File {
        path := filepath.Join(targetDir, file.Name)

        if file.FileInfo().IsDir() {
            if err := os.MkdirAll(path, 0755); err != nil {
                return "", fmt.Errorf("failed to create directory: %w", err)
            }
            continue
        }

        if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
            return "", fmt.Errorf("failed to create directory: %w", err)
        }

        outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
        if err != nil {
            return "", fmt.Errorf("failed to create file: %w", err)
        }

        rc, err := file.Open()
        if err != nil {
            outFile.Close()
            return "", fmt.Errorf("failed to open zip file: %w", err)
        }

        _, err = io.Copy(outFile, rc)
        outFile.Close()
        rc.Close()
        if err != nil {
            return "", fmt.Errorf("failed to extract file: %w", err)
        }
    }

    return targetDir, nil
} 