package testrunner

import "404skill-cli/testreport"

// TestRunner interface for running tests on projects
type TestRunner interface {
	RunTests(project Project, progressCallback func(string)) (*testreport.ParseResult, error)
}

// Project represents a project that can be tested
type Project struct {
	ID       string
	Name     string
	Language string
}
