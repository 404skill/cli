package filesystem

import (
	"os"
	"os/exec"
	"runtime"
)

// Manager handles file system operations
type Manager struct{}

// NewManager creates a new filesystem manager
func NewManager() *Manager {
	return &Manager{}
}

// OpenFileExplorer opens the file explorer at the specified path
func (f *Manager) OpenFileExplorer(path string) error {
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
func (f *Manager) CreateDirectory(path string) error {
	return os.MkdirAll(path, 0755)
}

// RemoveDirectory removes a directory and all its contents
func (f *Manager) RemoveDirectory(path string) error {
	return os.RemoveAll(path)
}

// DirectoryExists checks if a directory exists
func (f *Manager) DirectoryExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
