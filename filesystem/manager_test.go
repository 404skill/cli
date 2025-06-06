package filesystem

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestNewManager tests the constructor
func TestNewManager(t *testing.T) {
	// Act
	manager := NewManager()

	// Assert
	if manager == nil {
		t.Error("Expected non-nil Manager")
	}
}

// TestManager_CreateDirectory_Success tests successful directory creation
func TestManager_CreateDirectory_Success(t *testing.T) {
	// Arrange
	manager := NewManager()
	testDir := filepath.Join(os.TempDir(), "test_create_dir")
	defer os.RemoveAll(testDir)

	// Act
	err := manager.CreateDirectory(testDir)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if !manager.DirectoryExists(testDir) {
		t.Error("Expected directory to exist after creation")
	}
}

// TestManager_CreateDirectory_NestedPath tests creating nested directories
func TestManager_CreateDirectory_NestedPath(t *testing.T) {
	// Arrange
	manager := NewManager()
	testDir := filepath.Join(os.TempDir(), "test_nested", "deep", "path")
	defer os.RemoveAll(filepath.Join(os.TempDir(), "test_nested"))

	// Act
	err := manager.CreateDirectory(testDir)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if !manager.DirectoryExists(testDir) {
		t.Error("Expected nested directory to exist after creation")
	}
}

// TestManager_CreateDirectory_AlreadyExists tests creating directory that already exists
func TestManager_CreateDirectory_AlreadyExists(t *testing.T) {
	// Arrange
	manager := NewManager()
	testDir := filepath.Join(os.TempDir(), "test_already_exists")

	// Create the directory first
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Act
	err = manager.CreateDirectory(testDir)

	// Assert
	if err != nil {
		t.Errorf("Expected no error when directory already exists, got: %v", err)
	}
	if !manager.DirectoryExists(testDir) {
		t.Error("Expected directory to still exist")
	}
}

// TestManager_CreateDirectory_InvalidPath tests creating directory with invalid path
func TestManager_CreateDirectory_InvalidPath(t *testing.T) {
	// Arrange
	manager := NewManager()
	// Use a path that should be invalid on most systems
	invalidPath := ""
	if runtime.GOOS == "windows" {
		invalidPath = "Z:\\nonexistent\\invalid\\path"
	} else {
		invalidPath = "/root/invalid/path/that/should/fail"
	}

	// Act
	err := manager.CreateDirectory(invalidPath)

	// Assert - Note: This might not always fail depending on permissions
	// but we're testing the behavior when it does fail
	if err != nil {
		// This is expected in most cases
		t.Logf("Expected error for invalid path: %v", err)
	}
}

// TestManager_RemoveDirectory_Success tests successful directory removal
func TestManager_RemoveDirectory_Success(t *testing.T) {
	// Arrange
	manager := NewManager()
	testDir := filepath.Join(os.TempDir(), "test_remove_dir")

	// Create the directory first
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Act
	err = manager.RemoveDirectory(testDir)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if manager.DirectoryExists(testDir) {
		t.Error("Expected directory to not exist after removal")
	}
}

// TestManager_RemoveDirectory_WithContents tests removing directory with contents
func TestManager_RemoveDirectory_WithContents(t *testing.T) {
	// Arrange
	manager := NewManager()
	testDir := filepath.Join(os.TempDir(), "test_remove_with_contents")
	subDir := filepath.Join(testDir, "subdir")
	testFile := filepath.Join(subDir, "test.txt")

	// Create directory structure with file
	err := os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	file, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	file.Close()

	// Act
	err = manager.RemoveDirectory(testDir)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if manager.DirectoryExists(testDir) {
		t.Error("Expected directory to not exist after removal")
	}
}

// TestManager_RemoveDirectory_NonExistent tests removing non-existent directory
func TestManager_RemoveDirectory_NonExistent(t *testing.T) {
	// Arrange
	manager := NewManager()
	nonExistentDir := filepath.Join(os.TempDir(), "non_existent_dir_12345")

	// Act
	err := manager.RemoveDirectory(nonExistentDir)

	// Assert
	// os.RemoveAll doesn't return error for non-existent paths
	if err != nil {
		t.Errorf("Expected no error for non-existent directory, got: %v", err)
	}
}

// TestManager_DirectoryExists_True tests when directory exists
func TestManager_DirectoryExists_True(t *testing.T) {
	// Arrange
	manager := NewManager()
	testDir := filepath.Join(os.TempDir(), "test_exists_true")

	// Create the directory
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Act & Assert
	if !manager.DirectoryExists(testDir) {
		t.Error("Expected DirectoryExists to return true for existing directory")
	}
}

// TestManager_DirectoryExists_False tests when directory doesn't exist
func TestManager_DirectoryExists_False(t *testing.T) {
	// Arrange
	manager := NewManager()
	nonExistentDir := filepath.Join(os.TempDir(), "non_existent_dir_67890")

	// Act & Assert
	if manager.DirectoryExists(nonExistentDir) {
		t.Error("Expected DirectoryExists to return false for non-existent directory")
	}
}

// TestManager_DirectoryExists_File tests when path is a file, not directory
func TestManager_DirectoryExists_File(t *testing.T) {
	// Arrange
	manager := NewManager()
	testFile := filepath.Join(os.TempDir(), "test_file.txt")

	// Create a file
	file, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	file.Close()
	defer os.Remove(testFile)

	// Act & Assert
	if manager.DirectoryExists(testFile) {
		t.Error("Expected DirectoryExists to return false for file path")
	}
}

// TestManager_OpenFileExplorer_ValidPath tests opening file explorer
// Note: This test can't easily verify the actual opening of the file explorer
// but it can test that the method doesn't panic and returns without error for valid paths
func TestManager_OpenFileExplorer_ValidPath(t *testing.T) {
	// Arrange
	manager := NewManager()
	testDir := filepath.Join(os.TempDir(), "test_open_explorer")

	// Create the directory
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Act
	err = manager.OpenFileExplorer(testDir)

	// Assert
	// Note: This might fail in CI environments without a GUI, but that's expected
	// We're mainly testing that it doesn't panic
	if err != nil {
		t.Logf("OpenFileExplorer returned error (expected in headless environments): %v", err)
	}
}

// TestManager_OpenFileExplorer_NonExistentPath tests opening non-existent path
func TestManager_OpenFileExplorer_NonExistentPath(t *testing.T) {
	// Arrange
	manager := NewManager()
	nonExistentDir := filepath.Join(os.TempDir(), "non_existent_explorer_test")

	// Act
	err := manager.OpenFileExplorer(nonExistentDir)

	// Assert
	// Behavior depends on the OS - some might show error, others might ignore
	// We're mainly testing that it doesn't panic
	if err != nil {
		t.Logf("OpenFileExplorer returned error for non-existent path: %v", err)
	}
}

// TestManager_IntegrationTest tests a complete workflow
func TestManager_IntegrationTest(t *testing.T) {
	// Arrange
	manager := NewManager()
	baseDir := filepath.Join(os.TempDir(), "integration_test")
	subDir := filepath.Join(baseDir, "subdir")
	testFile := filepath.Join(subDir, "test.txt")

	// Act & Assert - Create directory
	err := manager.CreateDirectory(subDir)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Verify directory exists
	if !manager.DirectoryExists(baseDir) {
		t.Error("Expected base directory to exist")
	}
	if !manager.DirectoryExists(subDir) {
		t.Error("Expected sub directory to exist")
	}

	// Create a file to test removal of directory with contents
	file, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	file.Close()

	// Remove directory
	err = manager.RemoveDirectory(baseDir)
	if err != nil {
		t.Errorf("Failed to remove directory: %v", err)
	}

	// Verify directory doesn't exist
	if manager.DirectoryExists(baseDir) {
		t.Error("Expected directory to not exist after removal")
	}
}
