package testrunner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"404skill-cli/testreport"
)

// DefaultTestRunner implements TestRunner using docker-compose
type DefaultTestRunner struct{}

// NewDefaultTestRunner creates a new test runner
func NewDefaultTestRunner() *DefaultTestRunner {
	return &DefaultTestRunner{}
}

// RunTests executes tests for a project using docker-compose
func (r *DefaultTestRunner) RunTests(project Project, progressCallback func(string)) (*testreport.ParseResult, error) {
	projectDir, err := r.findProjectDirectory(project)
	if err != nil {
		return nil, fmt.Errorf("failed to find project directory: %w", err)
	}

	// Run docker-compose
	if err := r.runDockerCompose(projectDir, progressCallback); err != nil {
		return nil, fmt.Errorf("failed to run tests: %w", err)
	}

	// Parse test results
	result, err := r.parseTestResults(project, projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse test results: %w", err)
	}

	return result, nil
}

// findProjectDirectory locates the project directory in the user's home directory
func (r *DefaultTestRunner) findProjectDirectory(project Project) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	repo := strings.ToLower(strings.ReplaceAll(project.Name, " ", "_"))
	base := filepath.Join(home, "404skill_projects")

	entries, err := os.ReadDir(base)
	if err != nil {
		return "", fmt.Errorf("failed to read projects directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), repo) {
			return filepath.Join(base, entry.Name()), nil
		}
	}

	return "", fmt.Errorf("project directory not found for '%s'", repo)
}

// runDockerCompose executes docker-compose up with build and abort-on-container-exit flags
func (r *DefaultTestRunner) runDockerCompose(projectDir string, progressCallback func(string)) error {
	if progressCallback != nil {
		progressCallback("Starting docker-compose...")
	}

	cmd := exec.Command("docker", "compose", "up", "--build", "--abort-on-container-exit")
	cmd.Dir = projectDir

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker-compose failed: %w", err)
	}

	if progressCallback != nil {
		progressCallback("Tests completed")
	}

	return nil
}

// parseTestResults finds and parses the XML test report
func (r *DefaultTestRunner) parseTestResults(project Project, projectDir string) (*testreport.ParseResult, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	repo := strings.ToLower(strings.ReplaceAll(project.Name, " ", "_"))
	base := filepath.Join(home, "404skill_projects")

	reportsDir := filepath.Join(base, ".tests", fmt.Sprintf("%s_%s", repo, project.Language), "test-reports")

	entries, err := os.ReadDir(reportsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read reports directory: %w", err)
	}

	var xmlPath string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".xml") {
			xmlPath = filepath.Join(reportsDir, entry.Name())
			break
		}
	}

	if xmlPath == "" {
		return nil, fmt.Errorf("no XML test report found in %s", reportsDir)
	}

	parser := testreport.NewParser()
	result, err := parser.ParseFile(xmlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse test report: %w", err)
	}

	return result, nil
}
