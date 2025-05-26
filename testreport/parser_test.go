package testreport

import (
	"strings"
	"testing"
)

func TestParser_Parse(t *testing.T) {
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="TestSuite" tests="3" skipped="0" failures="1" errors="0" timestamp="2024-03-20T10:00:00" hostname="localhost" time="1.234">
  <testcase name="TestPassing" classname="TestSuite" time="0.5"/>
  <testcase name="TestFailing" classname="TestSuite" time="0.3">
    <failure message="Expected true but got false" type="AssertionError">Stack trace here</failure>
  </testcase>
  <testcase name="TestAnotherPassing" classname="TestSuite" time="0.434"/>
</testsuite>`

	parser := NewParser()
	result, err := parser.Parse(strings.NewReader(xmlContent))
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	// Verify test suite details
	if result.Suite.Name != "TestSuite" {
		t.Errorf("Expected suite name 'TestSuite', got '%s'", result.Suite.Name)
	}
	if result.Suite.Tests != 3 {
		t.Errorf("Expected 3 tests, got %d", result.Suite.Tests)
	}
	if result.Suite.Failures != 1 {
		t.Errorf("Expected 1 failure, got %d", result.Suite.Failures)
	}
	if result.Suite.Time != 1.234 {
		t.Errorf("Expected time 1.234, got %f", result.Suite.Time)
	}

	// Verify passed tests
	expectedPassed := []string{"TestPassing", "TestAnotherPassing"}
	if len(result.PassedTests) != len(expectedPassed) {
		t.Errorf("Expected %d passed tests, got %d", len(expectedPassed), len(result.PassedTests))
	}
	for i, name := range expectedPassed {
		if result.PassedTests[i] != name {
			t.Errorf("Expected passed test %d to be '%s', got '%s'", i, name, result.PassedTests[i])
		}
	}

	// Verify failed tests
	expectedFailed := []string{"TestFailing"}
	if len(result.FailedTests) != len(expectedFailed) {
		t.Errorf("Expected %d failed tests, got %d", len(expectedFailed), len(result.FailedTests))
	}
	for i, name := range expectedFailed {
		if result.FailedTests[i] != name {
			t.Errorf("Expected failed test %d to be '%s', got '%s'", i, name, result.FailedTests[i])
		}
	}

	// Verify test results
	if len(result.Suite.Results) != 3 {
		t.Errorf("Expected 3 test results, got %d", len(result.Suite.Results))
	}

	// Verify failing test details
	failingTest := result.Suite.Results[1]
	if failingTest.Name != "TestFailing" {
		t.Errorf("Expected failing test name 'TestFailing', got '%s'", failingTest.Name)
	}
	if failingTest.Failure == nil {
		t.Error("Expected failure details for failing test")
	} else {
		if failingTest.Failure.Message != "Expected true but got false" {
			t.Errorf("Expected failure message 'Expected true but got false', got '%s'", failingTest.Failure.Message)
		}
		if failingTest.Failure.Type != "AssertionError" {
			t.Errorf("Expected failure type 'AssertionError', got '%s'", failingTest.Failure.Type)
		}
	}
}

func TestParser_Parse_InvalidXML(t *testing.T) {
	parser := NewParser()
	_, err := parser.Parse(strings.NewReader("invalid xml"))
	if err == nil {
		t.Error("Expected error for invalid XML, got nil")
	}
}

func TestParser_Parse_InvalidTimestamp(t *testing.T) {
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="TestSuite" tests="1" skipped="0" failures="0" errors="0" timestamp="invalid-timestamp" hostname="localhost" time="1.0">
  <testcase name="TestPassing" classname="TestSuite" time="0.5"/>
</testsuite>`

	parser := NewParser()
	_, err := parser.Parse(strings.NewReader(xmlContent))
	if err == nil {
		t.Error("Expected error for invalid timestamp, got nil")
	}
}
