package language

import "404skill-cli/api"

// LanguageSelectedMsg is sent when a language is selected and download starts
type LanguageSelectedMsg struct {
	Project  *api.Project
	Language string
}

// DownloadCompleteMsg is sent when the download is completed successfully
type DownloadCompleteMsg struct {
	Project   *api.Project
	Language  string
	Directory string
}

// DownloadProgressMsg contains the current progress of the download operation
type DownloadProgressMsg struct {
	Progress float64
}

// DownloadErrorMsg is sent when the download fails
type DownloadErrorMsg struct {
	Error string
}
