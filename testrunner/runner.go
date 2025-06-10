package testrunner

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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

	// Parse test results - this will verify tests actually ran
	result, err := r.parseTestResults(project, projectDir)
	if err != nil {
		// If no test report found, docker-compose may have failed silently
		return nil, fmt.Errorf("tests may not have run properly - no recent test report found: %w", err)
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

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if progressCallback != nil {
		progressCallback(fmt.Sprintf("Running: docker compose up --build --abort-on-container-exit"))
		progressCallback(fmt.Sprintf("Working directory: %s", projectDir))
	}

	cmd.Run()
	exitCode := cmd.ProcessState.ExitCode()

	if progressCallback != nil {
		progressCallback(fmt.Sprintf("Docker-compose finished with exit code: %d", exitCode))
		if stdout.Len() > 0 {
			progressCallback(fmt.Sprintf("STDOUT: %s", stdout.String()))
		}
		if stderr.Len() > 0 {
			progressCallback(fmt.Sprintf("STDERR: %s", stderr.String()))
		}
	}

	// Exit code 0 = all tests passed
	// Exit code 1 = tests ran, but some failed (this is normal!)
	// Other exit codes = actual docker-compose failure
	if exitCode != 0 && exitCode != 1 {
		errorMsg := fmt.Sprintf("docker-compose failed with exit code %d", exitCode)
		if stdout.Len() > 0 {
			errorMsg += fmt.Sprintf("\n\n--- STDOUT ---\n%s", stdout.String())
		}
		if stderr.Len() > 0 {
			errorMsg += fmt.Sprintf("\n\n--- STDERR ---\n%s", stderr.String())
		}
		return fmt.Errorf("%s", errorMsg)
	}

	if progressCallback != nil {
		if exitCode == 0 {
			progressCallback("✅ All tests passed!")
		} else {
			progressCallback("⚠️  Tests completed - some may have failed")
		}
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
	var mostRecentTime time.Time

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".xml") {
			fullPath := filepath.Join(reportsDir, entry.Name())
			info, err := entry.Info()
			if err != nil {
				continue
			}

			// Find the most recent XML file
			if info.ModTime().After(mostRecentTime) {
				mostRecentTime = info.ModTime()
				xmlPath = fullPath
			}
		}
	}

	if xmlPath == "" {
		return nil, fmt.Errorf("no XML test report found in %s", reportsDir)
	}

	// Check if the test report is recent (within last 5 minutes)
	// This confirms tests actually ran and weren't just old files
	if time.Since(mostRecentTime) > 5*time.Minute {
		return nil, fmt.Errorf("test report found but is too old (%v) - tests may not have run", mostRecentTime)
	}

	parser := testreport.NewParser()
	result, err := parser.ParseFile(xmlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse test report: %w", err)
	}

	return result, nil
}
