package testreport

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"time"
)

// XMLTestSuites represents the XML structure of multiple test suites
type XMLTestSuites struct {
	XMLName    xml.Name       `xml:"testsuites"`
	TestSuites []XMLTestSuite `xml:"testsuite"`
}

// XMLTestSuite represents the XML structure of a test suite
type XMLTestSuite struct {
	XMLName   xml.Name      `xml:"testsuite"`
	Name      string        `xml:"name,attr"`
	Tests     int           `xml:"tests,attr"`
	Skipped   int           `xml:"skipped,attr"`
	Failures  int           `xml:"failures,attr"`
	Errors    int           `xml:"errors,attr"`
	Timestamp string        `xml:"timestamp,attr"`
	Hostname  string        `xml:"hostname,attr"`
	Time      float64       `xml:"time,attr"`
	TestCases []XMLTestCase `xml:"testcase"`
}

// XMLTestCase represents the XML structure of a test case
type XMLTestCase struct {
	Name      string      `xml:"name,attr"`
	ClassName string      `xml:"classname,attr"`
	Time      float64     `xml:"time,attr"`
	Failure   *XMLFailure `xml:"failure,omitempty"`
}

// XMLFailure represents the XML structure of a test failure
type XMLFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

// Parser handles parsing of test report XML files
type Parser struct{}

// NewParser creates a new test report parser
func NewParser() *Parser {
	return &Parser{}
}

// Parse reads and parses a test report from the given reader
func (p *Parser) Parse(reader io.Reader) (*ParseResult, error) {
	// Read all content first so we can try multiple parsing approaches
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read XML content: %w", err)
	}

	// First, try to parse as testsuites (multiple test suites)
	var xmlSuites XMLTestSuites
	if err := xml.NewDecoder(bytes.NewReader(content)).Decode(&xmlSuites); err == nil && len(xmlSuites.TestSuites) > 0 {
		// Successfully parsed as testsuites, use the first test suite
		return p.parseTestSuite(&xmlSuites.TestSuites[0])
	}

	// If that fails, try to parse as a single testsuite
	var xmlSuite XMLTestSuite
	if err := xml.NewDecoder(bytes.NewReader(content)).Decode(&xmlSuite); err != nil {
		return nil, fmt.Errorf("failed to decode XML: %w", err)
	}

	return p.parseTestSuite(&xmlSuite)
}

// parseTestSuite converts an XMLTestSuite to our domain model
func (p *Parser) parseTestSuite(xmlSuite *XMLTestSuite) (*ParseResult, error) {
	// Parse timestamp
	timestamp, err := time.Parse("2006-01-02T15:04:05", xmlSuite.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	// Convert XML suite to our domain model
	suite := TestSuite{
		Name:      xmlSuite.Name,
		Tests:     xmlSuite.Tests,
		Skipped:   xmlSuite.Skipped,
		Failures:  xmlSuite.Failures,
		Errors:    xmlSuite.Errors,
		Timestamp: timestamp,
		Hostname:  xmlSuite.Hostname,
		Time:      xmlSuite.Time,
		Results:   make([]TestResult, 0, len(xmlSuite.TestCases)),
	}

	// Process test cases
	passedTests := make([]string, 0)
	failedTests := make([]string, 0)

	for _, tc := range xmlSuite.TestCases {
		result := TestResult{
			Name:      tc.Name,
			ClassName: tc.ClassName,
			Time:      tc.Time,
			Passed:    tc.Failure == nil,
		}

		if tc.Failure != nil {
			result.Failure = &TestFailure{
				Message: tc.Failure.Message,
				Type:    tc.Failure.Type,
				Content: tc.Failure.Content,
			}
			failedTests = append(failedTests, tc.Name)
		} else {
			passedTests = append(passedTests, tc.Name)
		}

		suite.Results = append(suite.Results, result)
	}

	return &ParseResult{
		PassedTests:    passedTests,
		FailedTests:    failedTests,
		Suite:          suite,
		GroupedResults: p.groupTestsByTask(suite.Results),
	}, nil
}

// ParseFile parses a test report from a file
func (p *Parser) ParseFile(filename string) (*ParseResult, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return p.Parse(bytes.NewReader(file))
}

// extractTaskNumber extracts task number from various classname formats
// Supports formats like:
// - "test_api.TestTask1HealthCheck"
// - "Task 1: Health Check Endpoint"
// - "Task1Something"
// - "task_2_description"
func (p *Parser) extractTaskNumber(className string) int {
	// Define regex patterns for different task number formats
	patterns := []string{
		`(?i)testtask(\d+)`, // TestTask1, testtask2, etc.
		`(?i)task\s*(\d+)`,  // Task 1, task 2, Task1, etc.
		`(?i)task[_-](\d+)`, // task_1, task-2, etc.
		`(?i)(\d+).*task`,   // 1_task, 2-task, etc.
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(className)
		if len(matches) > 1 {
			taskNum, err := strconv.Atoi(matches[1])
			if err == nil {
				return taskNum
			}
		}
	}

	return -1 // No task number found
}

// groupTestsByTask groups tests by their task number
func (p *Parser) groupTestsByTask(results []TestResult) *GroupedTestResults {
	taskMap := make(map[int][]TestResult)

	// Group tests by task number
	for _, result := range results {
		taskNum := p.extractTaskNumber(result.ClassName)
		if taskNum == -1 {
			taskNum = 0 // Put tests without task numbers in "Task 0"
		}
		taskMap[taskNum] = append(taskMap[taskNum], result)
	}

	// Convert to TestClass structs and sort by task number
	var classes []TestClass
	var taskNumbers []int
	for taskNum := range taskMap {
		taskNumbers = append(taskNumbers, taskNum)
	}
	sort.Ints(taskNumbers)

	totalTests := 0
	totalPassed := 0
	totalFailed := 0
	totalTime := 0.0

	for _, taskNum := range taskNumbers {
		tests := taskMap[taskNum]

		var taskName string
		var displayName string
		if taskNum == 0 {
			taskName = "Uncategorized"
			displayName = "Uncategorized Tests"
		} else {
			taskName = fmt.Sprintf("Task%d", taskNum)
			displayName = fmt.Sprintf("Task %d", taskNum)
		}

		class := TestClass{
			Name:        taskName,
			DisplayName: displayName,
			Tests:       tests,
		}

		// Calculate statistics
		for _, test := range tests {
			class.TotalTime += test.Time
			if test.Passed {
				class.PassedCount++
				totalPassed++
			} else {
				class.FailedCount++
				totalFailed++
			}
			totalTests++
		}
		totalTime += class.TotalTime

		classes = append(classes, class)
	}

	return &GroupedTestResults{
		Classes:     classes,
		TotalTests:  totalTests,
		TotalPassed: totalPassed,
		TotalFailed: totalFailed,
		TotalTime:   totalTime,
	}
}
