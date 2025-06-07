package tui

import (
	"404skill-cli/api"
	"404skill-cli/tui/components/menu"
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// LanguageComponent handles language selection and project cloning
type LanguageComponent struct {
	project       *api.Project
	menu          *menu.Component
	cloning       bool
	progress      float64
	errorMsg      string
	fileManager   FileManager
	configManager ConfigManager
}

// NewLanguageComponent creates a new language component
func NewLanguageComponent(project *api.Project, fileManager FileManager, configManager ConfigManager) *LanguageComponent {
	languages := strings.Split(project.Language, ",")
	for i := range languages {
		languages[i] = strings.TrimSpace(languages[i])
	}

	// Create menu with custom styles to match existing theme
	languageMenu := menu.New(languages)
	menuStyles := menu.DefaultStyles()
	menuStyles.ItemStyle = menuItemStyle
	menuStyles.SelectedStyle = selectedMenuItemStyle
	languageMenu.SetStyles(menuStyles)

	return &LanguageComponent{
		project:       project,
		menu:          languageMenu,
		fileManager:   fileManager,
		configManager: configManager,
	}
}

// Init initializes the language component
func (l *LanguageComponent) Init() tea.Cmd {
	return nil
}

// Update handles messages for the language component
func (l *LanguageComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	var cmd tea.Cmd

	// Update menu component
	if !l.cloning {
		l.menu, cmd = l.menu.Update(msg)
	}

	switch msg := msg.(type) {
	case menu.MenuSelectMsg:
		if l.project != nil {
			l.cloning = true
			l.errorMsg = ""
			return l, l.cloneProject(l.project.Name, msg.SelectedItem)
		}
	case cloneCompleteMsg:
		l.cloning = false
		return l, tea.Batch(
			func() tea.Msg { return languageSelectedMsg{language: l.menu.GetSelectedItem()} },
		)
	case cloneProgressMsg:
		l.progress = msg.progress
		return l, nil
	case errMsg:
		l.errorMsg = msg.err.Error()
		l.cloning = false
		return l, nil
	}

	return l, cmd
}

// View renders the language component
func (l *LanguageComponent) View() string {
	if l.cloning {
		progress := int(l.progress * 100)
		progressBar := strings.Repeat("█", progress/10) + strings.Repeat("░", 10-progress/10)
		return fmt.Sprintf("%s\n\nCloning project...\n[%s] %d%%\n\nPress q to quit",
			headerStyle.Render("Cloning Project"),
			progressBar,
			progress)
	}

	view := headerStyle.Render("\nSelect a language for "+l.project.Name) + "\n\n"
	view += l.menu.View()
	view += "\n" + helpStyle.Render("Use ↑/↓ or k/j to move, Enter to select, [esc/b] back, q to quit")

	if l.errorMsg != "" {
		view += "\n\n" + errorStyle.Render("Error: "+l.errorMsg)
	}
	return view
}

// cloneProject initiates the git clone operation
func (l *LanguageComponent) cloneProject(projectName, language string) tea.Cmd {
	return func() tea.Msg {
		// Create projects directory if it doesn't exist
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to get home directory: %w", err)}
		}

		projectsDir := filepath.Join(homeDir, "404skill_projects")
		if err := l.fileManager.CreateDirectory(projectsDir); err != nil {
			return errMsg{err: fmt.Errorf("failed to create projects directory: %w", err)}
		}

		// Format project name for repo URL
		repoName := strings.ToLower(strings.ReplaceAll(projectName, " ", "_"))
		repoURL := fmt.Sprintf("https://github.com/404skill/%s_%s", repoName, language)
		targetDir := filepath.Join(projectsDir, fmt.Sprintf("%s_%s", repoName, language))

		// Remove existing directory if it exists
		if err := l.fileManager.RemoveDirectory(targetDir); err != nil {
			return errMsg{err: fmt.Errorf("failed to remove existing directory: %w", err)}
		}

		// Start git clone with progress output
		cmd := exec.Command("git", "clone", "--progress", repoURL, targetDir)
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to create stderr pipe: %w", err)}
		}

		if err := cmd.Start(); err != nil {
			return errMsg{err: fmt.Errorf("failed to start git clone: %w", err)}
		}

		// Read progress from stderr
		scanner := bufio.NewScanner(stderr)
		var cloneError string
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "Receiving objects") {
				// Parse percentage from line like "Receiving objects: 45% (9/20)"
				if strings.Contains(line, "%") {
					parts := strings.Split(line, "%")
					if len(parts) > 0 {
						if progress, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64); err == nil {
							// Send progress update
							tea.Batch(
								func() tea.Msg { return cloneProgressMsg{progress: progress / 100} },
							)()
						}
					}
				}
			} else if strings.Contains(line, "error:") || strings.Contains(line, "fatal:") {
				cloneError = line
			}
		}

		if err := cmd.Wait(); err != nil {
			if cloneError != "" {
				return errMsg{err: fmt.Errorf("git clone failed: %s", cloneError)}
			}
			return errMsg{err: fmt.Errorf("git clone failed: %w", err)}
		}

		// Verify the clone was successful
		if !l.fileManager.DirectoryExists(targetDir) {
			return errMsg{err: fmt.Errorf("clone appeared to succeed but target directory is missing")}
		}

		// Update config with downloaded project
		if err := l.configManager.UpdateDownloadedProject(l.project.ID); err != nil {
			return errMsg{err: fmt.Errorf("failed to update config: %w", err)}
		}

		// Open file explorer at the cloned directory
		if err := l.fileManager.OpenFileExplorer(targetDir); err != nil {
			fmt.Printf("Warning: Failed to open file explorer: %v\n", err)
		}

		return cloneCompleteMsg{}
	}
}

// languageSelectedMsg is sent when a language is selected
type languageSelectedMsg struct {
	language string
}
