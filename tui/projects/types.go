package projects

import "404skill-cli/api"

// ProjectSelectedMsg is sent when a project is selected for download
type ProjectSelectedMsg struct {
	Project *api.Project
}

// ProjectsErrorMsg is sent when an error occurs in the projects component
type ProjectsErrorMsg struct {
	Error string
}

// ProjectRedownloadNeededMsg is sent when a downloaded project's directory is missing
type ProjectRedownloadNeededMsg struct {
	Project *api.Project
}

// ProjectOpenedMsg is sent when a downloaded project is successfully opened
type ProjectOpenedMsg struct {
	Message string
}
