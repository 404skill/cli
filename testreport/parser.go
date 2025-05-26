package testreport

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"time"
)

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
	var xmlSuite XMLTestSuite
	if err := xml.NewDecoder(reader).Decode(&xmlSuite); err != nil {
		return nil, fmt.Errorf("failed to decode XML: %w", err)
	}

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
			}
			failedTests = append(failedTests, tc.Name)
		} else {
			passedTests = append(passedTests, tc.Name)
		}

		suite.Results = append(suite.Results, result)
	}

	return &ParseResult{
		PassedTests: passedTests,
		FailedTests: failedTests,
		Suite:       suite,
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
