package testrunner

import (
	"regexp"
	"strings"
)

// LogFilter handles filtering of test execution logs
type LogFilter struct {
	dockerBuildPatterns   []*regexp.Regexp
	meaningfulPatterns    []*regexp.Regexp
	universalErrorWords   []string
	universalSuccessWords []string
}

// NewLogFilter creates a new log filter with predefined patterns
func NewLogFilter() *LogFilter {
	// Patterns for Docker build noise (to hide)
	dockerBuildPatterns := []*regexp.Regexp{
		regexp.MustCompile(`^#\d+`),                                        // Docker build steps
		regexp.MustCompile(`CACHED|DONE \d+\.\d+s`),                        // Docker layer status
		regexp.MustCompile(`exporting layers|writing image`),               // Image operations
		regexp.MustCompile(`transferring context|transferring dockerfile`), // File operations
		regexp.MustCompile(`FromAsCasing|WARN.*line \d+`),                  // Docker warnings
		regexp.MustCompile(`internal\] load|auth\]|resolving provenance`),  // Docker internals
		regexp.MustCompile(`pull token for registry`),                      // Registry operations
		regexp.MustCompile(`Container .* Recreat`),                         // Container recreation
		regexp.MustCompile(`Attaching to`),                                 // Docker compose attachment
	}

	// Patterns for meaningful content (to show)
	meaningfulPatterns := []*regexp.Regexp{
		regexp.MustCompile(`> Task :`),                    // Build system tasks
		regexp.MustCompile(`BUILD (SUCCESSFUL|FAILED)`),   // Build results
		regexp.MustCompile(`\d+ actionable tasks:`),       // Task summary
		regexp.MustCompile(`exited with code`),            // Exit status
		regexp.MustCompile(`(Starting|Stopping|Stopped)`), // Container lifecycle
	}

	return &LogFilter{
		dockerBuildPatterns: dockerBuildPatterns,
		meaningfulPatterns:  meaningfulPatterns,
		universalErrorWords: []string{
			"ERROR", "FAILED", "FAIL", "Exception", "Error:",
			"WARN", "WARNING", "Fatal", "Critical",
		},
		universalSuccessWords: []string{
			"BUILD SUCCESSFUL", "PASSED", "SUCCESS", "✅", "✓",
			"All tests", "completed successfully",
		},
	}
}

// FilterLevel represents the level of filtering to apply
type FilterLevel int

const (
	FilterNone    FilterLevel = iota // Show everything (verbose mode)
	FilterBasic                      // Hide Docker noise, show meaningful content
	FilterMinimal                    // Show only high-level status
)

// FilteredMessage represents a processed log message
type FilteredMessage struct {
	Original   string
	Filtered   string
	Level      MessageLevel
	ShouldShow bool
}

// MessageLevel categorizes the importance of a message
type MessageLevel int

const (
	LevelNoise   MessageLevel = iota // Docker build noise
	LevelInfo                        // General information
	LevelTask                        // Build/test tasks
	LevelStatus                      // Important status updates
	LevelError                       // Errors and warnings
	LevelSuccess                     // Success messages
)

// FilterMessage processes a log line and determines how to display it
func (f *LogFilter) FilterMessage(message string, filterLevel FilterLevel) FilteredMessage {
	trimmed := strings.TrimSpace(message)

	result := FilteredMessage{
		Original: message,
		Filtered: trimmed,
		Level:    f.categorizeMessage(trimmed),
	}

	switch filterLevel {
	case FilterNone:
		result.ShouldShow = true
	case FilterBasic:
		result.ShouldShow = f.shouldShowInBasicMode(result.Level)
	case FilterMinimal:
		result.ShouldShow = f.shouldShowInMinimalMode(result.Level)
	}

	return result
}

// categorizeMessage determines the level/importance of a message
func (f *LogFilter) categorizeMessage(message string) MessageLevel {
	// Check for errors first (highest priority)
	for _, errorWord := range f.universalErrorWords {
		if strings.Contains(strings.ToUpper(message), strings.ToUpper(errorWord)) {
			return LevelError
		}
	}

	// Check for success indicators
	for _, successWord := range f.universalSuccessWords {
		if strings.Contains(strings.ToUpper(message), strings.ToUpper(successWord)) {
			return LevelSuccess
		}
	}

	// Check for meaningful patterns
	for _, pattern := range f.meaningfulPatterns {
		if pattern.MatchString(message) {
			return LevelTask
		}
	}

	// Check for Docker build noise
	for _, pattern := range f.dockerBuildPatterns {
		if pattern.MatchString(message) {
			return LevelNoise
		}
	}

	// Default to info level
	return LevelInfo
}

// shouldShowInBasicMode determines if a message should be shown in basic filtering mode
func (f *LogFilter) shouldShowInBasicMode(level MessageLevel) bool {
	switch level {
	case LevelNoise:
		return false // Hide Docker noise
	case LevelInfo, LevelTask, LevelStatus, LevelError, LevelSuccess:
		return true // Show everything else
	default:
		return true
	}
}

// shouldShowInMinimalMode determines if a message should be shown in minimal filtering mode
func (f *LogFilter) shouldShowInMinimalMode(level MessageLevel) bool {
	switch level {
	case LevelNoise, LevelInfo:
		return false // Hide noise and general info
	case LevelTask, LevelStatus, LevelError, LevelSuccess:
		return true // Show only important updates
	default:
		return false
	}
}

// GetHighLevelStatus extracts a high-level status from a message
func (f *LogFilter) GetHighLevelStatus(message string) string {
	filtered := f.FilterMessage(message, FilterBasic)

	switch filtered.Level {
	case LevelTask:
		if strings.Contains(message, "> Task :test") {
			if strings.Contains(message, "UP-TO-DATE") {
				return "Tests are up-to-date"
			} else if strings.Contains(message, "NO-SOURCE") {
				return "No test sources found"
			} else {
				return "Running tests..."
			}
		}
		if strings.Contains(message, "> Task :build") {
			return "Building project..."
		}
		if strings.Contains(message, "> Task :compile") {
			return "Compiling sources..."
		}
	case LevelSuccess:
		if strings.Contains(message, "BUILD SUCCESSFUL") {
			return "✅ Build completed successfully"
		}
		return "✅ Success"
	case LevelError:
		return "❌ Error occurred"
	}

	return ""
}
