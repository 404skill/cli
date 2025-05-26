# Test Report Parser

This package provides functionality to parse XML test reports and extract information about passed and failed tests.

## Features

- Parse XML test reports from files or readers
- Extract lists of passed and failed tests
- Access detailed test suite information
- Handle test failures with messages and types
- Comprehensive test coverage

## Usage

```go
package main

import (
    "fmt"
    "log"
    "404skill-cli/testreport"
)

func main() {
    parser := testreport.NewParser()
    
    // Parse from a file
    result, err := parser.ParseFile("test-results.xml")
    if err != nil {
        log.Fatal(err)
    }
    
    // Access passed and failed tests
    fmt.Printf("Passed tests: %v\n", result.PassedTests)
    fmt.Printf("Failed tests: %v\n", result.FailedTests)
    
    // Access test suite details
    suite := result.Suite
    fmt.Printf("Suite name: %s\n", suite.Name)
    fmt.Printf("Total tests: %d\n", suite.Tests)
    fmt.Printf("Failures: %d\n", suite.Failures)
    
    // Access individual test results
    for _, test := range suite.Results {
        fmt.Printf("Test: %s, Passed: %v\n", test.Name, test.Passed)
        if !test.Passed {
            fmt.Printf("Failure message: %s\n", test.Failure.Message)
        }
    }
}
```

## XML Format

The parser expects XML test reports in the following format:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="TestSuite" tests="3" skipped="0" failures="1" errors="0" timestamp="2024-03-20T10:00:00" hostname="localhost" time="1.234">
  <testcase name="TestPassing" classname="TestSuite" time="0.5"/>
  <testcase name="TestFailing" classname="TestSuite" time="0.3">
    <failure message="Expected true but got false" type="AssertionError">Stack trace here</failure>
  </testcase>
  <testcase name="TestAnotherPassing" classname="TestSuite" time="0.434"/>
</testsuite>
```

## Types

### TestResult
Represents a single test case result with the following fields:
- `Name`: The name of the test
- `ClassName`: The class name of the test
- `Time`: Execution time in seconds
- `Passed`: Whether the test passed
- `Failure`: Details about the failure (if any)

### TestFailure
Represents a test failure with:
- `Message`: The failure message
- `Type`: The type of failure

### TestSuite
Represents a complete test suite with:
- `Name`: Suite name
- `Tests`: Total number of tests
- `Skipped`: Number of skipped tests
- `Failures`: Number of failed tests
- `Errors`: Number of errors
- `Timestamp`: When the tests were run
- `Hostname`: The host where tests were run
- `Time`: Total execution time
- `Results`: List of individual test results

### ParseResult
The result of parsing a test report containing:
- `PassedTests`: List of names of passed tests
- `FailedTests`: List of names of failed tests
- `Suite`: The complete test suite information 