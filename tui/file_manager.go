package tui

import (
	"os"
	"os/exec"
	"runtime"
)

// DefaultFileManager implements the FileManager interface
type DefaultFileManager struct{}

// NewDefaultFileManager creates a new default file manager
func NewDefaultFileManager() *DefaultFileManager {
	return &DefaultFileManager{}
}

// OpenFileExplorer opens the file explorer at the specified path
func (f *DefaultFileManager) OpenFileExplorer(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}

// CreateDirectory creates a directory if it doesn't exist
func (f *DefaultFileManager) CreateDirectory(path string) error {
	return os.MkdirAll(path, 0755)
}

// RemoveDirectory removes a directory and all its contents
func (f *DefaultFileManager) RemoveDirectory(path string) error {
	return os.RemoveAll(path)
}

// DirectoryExists checks if a directory exists
func (f *DefaultFileManager) DirectoryExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
