package testreport

import "time"

// TestResult represents the result of a single test case
type TestResult struct {
	Name      string
	ClassName string
	Time      float64
	Passed    bool
	Failure   *TestFailure
}

// TestFailure represents a test failure with its message and type
type TestFailure struct {
	Message string
	Type    string
}

// TestSuite represents a complete test suite with its results
type TestSuite struct {
	Name      string
	Tests     int
	Skipped   int
	Failures  int
	Errors    int
	Timestamp time.Time
	Hostname  string
	Time      float64
	Results   []TestResult
}

// ParseResult represents the result of parsing a test report
type ParseResult struct {
	PassedTests []string
	FailedTests []string
	Suite       TestSuite
}
