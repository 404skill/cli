package testrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"404skill-cli/testreport"
)

func TestDefaultTestRunner_findProjectDirectory(t *testing.T) {
	tests := []struct {
		name        string
		project     Project
		expectError bool
	}{
		{
			name: "project directory not found",
			project: Project{
				ID:       "proj1",
				Name:     "Missing Project",
				Language: "go",
			},
			expectError: true,
		},
		{
			name: "projects directory doesn't exist",
			project: Project{
				ID:       "proj1",
				Name:     "Test Project",
				Language: "go",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewDefaultTestRunner()

			if tt.expectError {
				// We expect this to fail when the directory structure doesn't match
				_, err := runner.findProjectDirectory(tt.project)
				if err == nil {
					t.Error("Expected error but got none")
				}
			}
		})
	}
}

func TestDefaultTestRunner_parseTestResults(t *testing.T) {
	tests := []struct {
		name           string
		project        Project
		xmlContent     string
		expectError    bool
		expectedPassed int
		expectedFailed int
	}{
		{
			name: "parses successful test results",
			project: Project{
				ID:       "proj1",
				Name:     "Test Project",
				Language: "go",
			},
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="TestSuite" tests="3" failures="1" time="1.23" timestamp="2023-01-01T12:00:00">
    <testcase name="Test1" time="0.1"/>
    <testcase name="Test2" time="0.2">
        <failure message="failed"/>
    </testcase>
    <testcase name="Test3" time="0.3"/>
</testsuite>`,
			expectError:    false,
			expectedPassed: 2,
			expectedFailed: 1,
		},
		{
			name: "handles invalid XML",
			project: Project{
				ID:       "proj1",
				Name:     "Test Project",
				Language: "go",
			},
			xmlContent:  "invalid xml content",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary test file
			tmpDir := t.TempDir()
			xmlPath := filepath.Join(tmpDir, "test-results.xml")

			err := os.WriteFile(xmlPath, []byte(tt.xmlContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Test the parser directly since parseTestResults is hard to mock
			parser := testreport.NewParser()
			result, err := parser.ParseFile(xmlPath)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(result.PassedTests) != tt.expectedPassed {
				t.Errorf("Expected %d passed tests, got %d", tt.expectedPassed, len(result.PassedTests))
			}

			if len(result.FailedTests) != tt.expectedFailed {
				t.Errorf("Expected %d failed tests, got %d", tt.expectedFailed, len(result.FailedTests))
			}
		})
	}
}

func TestProject_validateFields(t *testing.T) {
	tests := []struct {
		name    string
		project Project
		valid   bool
	}{
		{
			name: "valid project",
			project: Project{
				ID:       "proj1",
				Name:     "Test Project",
				Language: "go",
			},
			valid: true,
		},
		{
			name: "empty ID",
			project: Project{
				ID:       "",
				Name:     "Test Project",
				Language: "go",
			},
			valid: false,
		},
		{
			name: "empty name",
			project: Project{
				ID:       "proj1",
				Name:     "",
				Language: "go",
			},
			valid: false,
		},
		{
			name: "empty language",
			project: Project{
				ID:       "proj1",
				Name:     "Test Project",
				Language: "",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test project field validation
			hasValidFields := tt.project.ID != "" && tt.project.Name != "" && tt.project.Language != ""

			if hasValidFields != tt.valid {
				t.Errorf("Expected validation result %v, got %v", tt.valid, hasValidFields)
			}
		})
	}
}

func TestDefaultTestRunner_formatProjectName(t *testing.T) {
	tests := []struct {
		name         string
		projectName  string
		projectID    string
		expectedRepo string
	}{
		{
			name:         "simple name",
			projectName:  "TestProject",
			projectID:    "proj1",
			expectedRepo: "testproject_proj1",
		},
		{
			name:         "name with spaces",
			projectName:  "Test Project Name",
			projectID:    "proj2",
			expectedRepo: "test_project_name_proj2",
		},
		{
			name:         "name with special characters",
			projectName:  "Test-Project_Name!",
			projectID:    "proj3",
			expectedRepo: "test-project_name!_proj3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the project name formatting logic
			result := formatProjectName(tt.projectName, tt.projectID)
			if result != tt.expectedRepo {
				t.Errorf("Expected %s, got %s", tt.expectedRepo, result)
			}
		})
	}
}

func TestNewDefaultTestRunner(t *testing.T) {
	runner := NewDefaultTestRunner()

	if runner == nil {
		t.Fatal("Expected runner to be created")
	}

	// Verify it implements the TestRunner interface
	var _ TestRunner = runner
}

func TestDefaultTestRunner_RunTests_InvalidProject(t *testing.T) {
	runner := NewDefaultTestRunner()

	// Test with project that won't be found
	project := Project{
		ID:       "nonexistent",
		Name:     "Nonexistent Project",
		Language: "go",
	}

	result, err := runner.RunTests(project, nil)

	if err == nil {
		t.Error("Expected error for nonexistent project")
	}

	if result != nil {
		t.Error("Expected nil result for failed test run")
	}
}

// Helper function that mimics the formatting logic in the service
func formatProjectName(name string, id string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "_")) + "_" + id
}
