package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"404skill-cli/testreport"
	"404skill-cli/tui/testresults"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Create sample test results with some failures
	results := &testreport.ParseResult{
		Suite: testreport.TestSuite{
			Name:      "Sample Test Suite",
			Tests:     5,
			Failures:  2,
			Errors:    0,
			Time:      3.45,
			Timestamp: time.Now(),
		},
		PassedTests: []string{"test_user_login", "test_data_validation", "test_simple_calculation"},
		FailedTests: []string{"test_large_dataset_processing", "test_memory_intensive_operation"},
	}

	// Add detailed test results
	results.Suite.Results = []testreport.TestResult{
		{
			Name:   "test_user_login",
			Passed: true,
			Time:   0.25,
		},
		{
			Name:   "test_data_validation",
			Passed: true,
			Time:   0.18,
		},
		{
			Name:   "test_simple_calculation",
			Passed: true,
			Time:   0.05,
		},
		{
			Name:   "test_large_dataset_processing",
			Passed: false,
			Time:   2.15,
			Failure: &testreport.TestFailure{
				Message: "AssertionError: Expected 1000 records but got 999",
				Type:    "AssertionError",
				Content: `Traceback (most recent call last):
  File "test_processing.py", line 45, in test_large_dataset_processing
    assert len(processed_data) == 1000
AssertionError: Expected 1000 records but got 999

Additional context:
- Dataset size: 1000 records
- Memory usage: 512MB
- Processing time: 2.15s
- Last record index: 998`,
			},
		},
		{
			Name:   "test_memory_intensive_operation",
			Passed: false,
			Time:   0.82,
			Failure: &testreport.TestFailure{
				Message: "OutOfMemoryError: Unable to allocate 2GB for operation",
				Type:    "OutOfMemoryError",
				Content: `java.lang.OutOfMemoryError: Java heap space
	at com.example.processor.DataProcessor.processLargeArray(DataProcessor.java:123)
	at com.example.test.MemoryTest.test_memory_intensive_operation(MemoryTest.java:67)
	
Heap dump available at: /tmp/heap_dump_2024-01-15.hprof
Available memory: 1.5GB
Required memory: 2.0GB`,
			},
		},
	}

	// Create test results component
	component := testresults.New()
	component.SetResults(results)

	// Create Bubble Tea program
	p := tea.NewProgram(component, tea.WithAltScreen())

	fmt.Println("🧪 Enhanced Test Results Demo")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("This demo shows the new expandable test results component.")
	fmt.Println()
	fmt.Println("Navigation:")
	fmt.Println("  ↑/↓ or k/j  - Navigate tests")
	fmt.Println("  →/l         - Expand failed test details")
	fmt.Println("  ←/h         - Collapse failed test details")
	fmt.Println("  Enter/Space - Toggle expansion")
	fmt.Println("  Tab         - Cycle through failure sections")
	fmt.Println("  Esc/b       - Back (in real app)")
	fmt.Println("  q           - Quit")
	fmt.Println()
	fmt.Printf("Press Enter to start the demo...")
	fmt.Scanln()

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	fmt.Println("\n✅ Demo completed!")
	fmt.Println("\nFeatures demonstrated:")
	fmt.Println("• Enhanced test results with expandable failure details")
	fmt.Println("• Clear visual indicators for passed/failed tests")
	fmt.Println("• Detailed failure messages and stack traces")
	fmt.Println("• Intuitive keyboard navigation")
	fmt.Println("• Consistent styling with the existing application theme")
}
