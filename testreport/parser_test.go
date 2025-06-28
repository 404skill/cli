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

func TestParser_ExtractTaskNumber(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		className string
		expected  int
		name      string
	}{
		// Original TestTask format
		{"test_api.TestTask1HealthCheck", 1, "Original TestTask format"},
		{"test_api.TestTask2JournalEntry", 2, "TestTask with number 2"},
		{"test_api.TestTask10Advanced", 10, "TestTask with two digits"},
		{"test_api.TestTask123Multi", 123, "TestTask with three digits"},
		{"test_api.TestTask1000Large", 1000, "TestTask with four digits"},
		{"TestTask5Simple", 5, "Simple TestTask format"},

		// Jest format - "Task N: Description"
		{"Task 1: Health Check Endpoint", 1, "Jest format Task 1"},
		{"Task 2: Database Connection", 2, "Jest format Task 2"},
		{"Task 10: Advanced Features", 10, "Jest format two digits"},
		{"Task 123: Complex Test", 123, "Jest format three digits"},
		{"Task 999: Large Number", 999, "Jest format large number"},

		// Task without space
		{"Task1Something", 1, "Task without space"},
		{"Task25NoSpace", 25, "Task without space - two digits"},
		{"Task100NoSpace", 100, "Task without space - three digits"},

		// Underscore/hyphen separated
		{"task_1_description", 1, "Underscore separated lowercase"},
		{"task_10_advanced", 10, "Underscore separated two digits"},
		{"task_123_complex", 123, "Underscore separated three digits"},
		{"Task_5_Mixed_Case", 5, "Underscore separated mixed case"},
		{"task-2-hyphen", 2, "Hyphen separated"},
		{"task-99-hyphen", 99, "Hyphen separated two digits"},

		// Number first format
		{"1_task_health", 1, "Number first with underscore"},
		{"10_task_database", 10, "Number first two digits"},
		{"123_task_complex", 123, "Number first three digits"},
		{"2-task-hyphen", 2, "Number first with hyphen"},

		// Case variations
		{"TESTTASK7UPPER", 7, "All uppercase TestTask"},
		{"testtask8lower", 8, "All lowercase testtask"},
		{"TestTask9Mixed", 9, "Mixed case TestTask"},
		{"TASK 15: UPPERCASE", 15, "Uppercase Task format"},
		{"task 20: lowercase", 20, "Lowercase task format"},

		// Edge cases and failures
		{"SomeOtherClass", -1, "No task pattern"},
		{"TestTaskNoNumber", -1, "TestTask without number"},
		{"Task: No Number", -1, "Task without number"},
		{"TaskABC123", -1, "Task with letters before number"},
		{"", -1, "Empty string"},
		{"Task", -1, "Just Task word"},
		{"123", -1, "Just number"},
		{"NotATaskPattern", -1, "No matching pattern"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.extractTaskNumber(tt.className)
			if result != tt.expected {
				t.Errorf("extractTaskNumber(%q) = %d, want %d", tt.className, result, tt.expected)
			}
		})
	}
}

func TestParser_GroupTestsByTask(t *testing.T) {
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="Test Suite" tests="5" failures="2" errors="0" time="2.5" timestamp="2024-03-20T10:00:00" hostname="localhost">
  <testcase name="test_health_endpoint_returns_200_ok" classname="test_api.TestTask1HealthCheck" time="0.5"/>
  <testcase name="test_db_connection" classname="test_api.TestTask1DatabaseConnection" time="0.3">
    <failure message="Connection failed">DB timeout</failure>
  </testcase>
  <testcase name="test_create_entry" classname="test_api.TestTask2JournalEntryCreation" time="0.8"/>
  <testcase name="test_validate_entry" classname="test_api.TestTask2JournalEntryValidation" time="0.9">
    <failure message="Validation failed">Invalid format</failure>
  </testcase>
  <testcase name="test_uncategorized" classname="SomeOtherClass" time="0.1"/>
</testsuite>`

	parser := NewParser()
	result, err := parser.Parse(strings.NewReader(xmlContent))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check that grouped results were created
	if result.GroupedResults == nil {
		t.Fatal("GroupedResults should not be nil")
	}

	grouped := result.GroupedResults

	// Should have 3 groups: Uncategorized (0), Task1, Task2
	if len(grouped.Classes) != 3 {
		t.Errorf("Expected 3 groups, got %d", len(grouped.Classes))
	}

	// Check totals
	if grouped.TotalTests != 5 {
		t.Errorf("Expected 5 total tests, got %d", grouped.TotalTests)
	}
	if grouped.TotalPassed != 3 {
		t.Errorf("Expected 3 passed tests, got %d", grouped.TotalPassed)
	}
	if grouped.TotalFailed != 2 {
		t.Errorf("Expected 2 failed tests, got %d", grouped.TotalFailed)
	}

	// Check groups are sorted by task number (Uncategorized=0, Task1=1, Task2=2)
	expectedNames := []string{"Uncategorized", "Task1", "Task2"}
	expectedDisplayNames := []string{"Uncategorized Tests", "Task 1", "Task 2"}
	expectedTestCounts := []int{1, 2, 2}

	for i, expected := range expectedNames {
		if i >= len(grouped.Classes) {
			t.Errorf("Expected group %d to exist", i)
			continue
		}

		class := grouped.Classes[i]
		if class.Name != expected {
			t.Errorf("Group %d: expected name %s, got %s", i, expected, class.Name)
		}
		if class.DisplayName != expectedDisplayNames[i] {
			t.Errorf("Group %d: expected display name %s, got %s", i, expectedDisplayNames[i], class.DisplayName)
		}
		if len(class.Tests) != expectedTestCounts[i] {
			t.Errorf("Group %d: expected %d tests, got %d", i, expectedTestCounts[i], len(class.Tests))
		}
	}

	// Check specific group contents
	task1 := grouped.Classes[1] // Task1
	if task1.PassedCount != 1 {
		t.Errorf("Task1: expected 1 passed test, got %d", task1.PassedCount)
	}
	if task1.FailedCount != 1 {
		t.Errorf("Task1: expected 1 failed test, got %d", task1.FailedCount)
	}

	task2 := grouped.Classes[2] // Task2
	if task2.PassedCount != 1 {
		t.Errorf("Task2: expected 1 passed test, got %d", task2.PassedCount)
	}
	if task2.FailedCount != 1 {
		t.Errorf("Task2: expected 1 failed test, got %d", task2.FailedCount)
	}
}
