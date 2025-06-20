package testrunner

import (
	"bufio"
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

	// Create log file for this test run
	logFile, err := r.createLogFile(projectDir, project)
	if err != nil {
		if progressCallback != nil {
			progressCallback(fmt.Sprintf("Warning: Could not create log file: %v", err))
		}
	}
	defer func() {
		if logFile != nil {
			logFile.Close()
		}
	}()

	// Run docker-compose
	if err := r.runDockerCompose(projectDir, logFile, progressCallback); err != nil {
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
func (r *DefaultTestRunner) runDockerCompose(projectDir string, logFile *os.File, progressCallback func(string)) error {
	if progressCallback != nil {
		progressCallback("Starting docker-compose...")
	}

	cmd := exec.Command("docker", "compose", "up", "--build", "--abort-on-container-exit")
	cmd.Dir = projectDir

	if progressCallback != nil {
		progressCallback(fmt.Sprintf("Running: docker compose up --build --abort-on-container-exit"))
		progressCallback(fmt.Sprintf("Working directory: %s", projectDir))
	}

	// Log the command being run
	if logFile != nil {
		logFile.WriteString(fmt.Sprintf("Command: docker compose up --build --abort-on-container-exit\n"))
		logFile.WriteString(fmt.Sprintf("Working Directory: %s\n\n", projectDir))
		logFile.WriteString("=== OUTPUT ===\n")
	}

	// Create pipes to capture output in real-time
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start docker-compose: %w", err)
	}

	// Track if tests were actually executed
	testsExecuted := false
	testsUpToDate := false

	// Stream stdout in real-time
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if progressCallback != nil {
				progressCallback(fmt.Sprintf("OUT: %s", line))
			}
			if logFile != nil {
				logFile.WriteString(fmt.Sprintf("STDOUT: %s\n", line))
			}

			// Check if tests are running or up-to-date
			if strings.Contains(line, "> Task :test") {
				if strings.Contains(line, "UP-TO-DATE") {
					testsUpToDate = true
				} else {
					testsExecuted = true
				}
			}
		}
	}()

	// Stream stderr in real-time
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if progressCallback != nil {
				progressCallback(fmt.Sprintf("ERR: %s", line))
			}
			if logFile != nil {
				logFile.WriteString(fmt.Sprintf("STDERR: %s\n", line))
			}
		}
	}()

	// Wait for command to finish
	err = cmd.Wait()
	exitCode := cmd.ProcessState.ExitCode()

	if progressCallback != nil {
		progressCallback(fmt.Sprintf("Docker-compose finished with exit code: %d", exitCode))

		if testsUpToDate && !testsExecuted {
			progressCallback("⚠️  WARNING: Tests were UP-TO-DATE - no tests actually ran!")
			progressCallback("This usually means:")
			progressCallback("  1. No test files exist in the project")
			progressCallback("  2. Tests haven't changed since last run")
			progressCallback("  3. Gradle is using cached results")
		}
	}

	if logFile != nil {
		logFile.WriteString(fmt.Sprintf("\n=== COMMAND FINISHED ===\n"))
		logFile.WriteString(fmt.Sprintf("Exit Code: %d\n", exitCode))
		logFile.WriteString(fmt.Sprintf("Tests Executed: %t\n", testsExecuted))
		logFile.WriteString(fmt.Sprintf("Tests Up-To-Date: %t\n", testsUpToDate))
		logFile.WriteString(fmt.Sprintf("Finished: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	}

	// Exit code 0 = all tests passed
	// Exit code 1 = tests ran, but some failed (this is normal!)
	// Other exit codes = actual docker-compose failure
	if exitCode != 0 && exitCode != 1 {
		return fmt.Errorf("docker-compose failed with exit code %d", exitCode)
	}

	if progressCallback != nil {
		if exitCode == 0 {
			if testsExecuted {
				progressCallback("✅ All tests passed!")
			} else if testsUpToDate {
				progressCallback("⚠️  Tests were up-to-date - no new tests ran")
			} else {
				progressCallback("✅ Build completed successfully")
			}
		} else {
			progressCallback("⚠️  Tests completed - some may have failed")
		}
		if logFile != nil {
			progressCallback(fmt.Sprintf("📝 Full log saved to: %s", logFile.Name()))
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

// createLogFile creates a timestamped log file for the test run
func (r *DefaultTestRunner) createLogFile(projectDir string, project Project) (*os.File, error) {
	logsDir := filepath.Join(projectDir, "test-logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logFileName := fmt.Sprintf("test-run_%s_%s.log", project.Language, timestamp)
	logPath := filepath.Join(logsDir, logFileName)

	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	// Write header to log file
	header := fmt.Sprintf("=== Test Run Log ===\n")
	header += fmt.Sprintf("Project: %s (%s)\n", project.Name, project.Language)
	header += fmt.Sprintf("Directory: %s\n", projectDir)
	header += fmt.Sprintf("Started: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	header += fmt.Sprintf("Log File: %s\n", logPath)
	header += fmt.Sprintf("========================\n\n")

	logFile.WriteString(header)
	return logFile, nil
}
