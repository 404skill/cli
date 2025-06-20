package variant

import (
	"404skill-cli/api"
	"404skill-cli/config"
	"404skill-cli/downloader"
	"404skill-cli/filesystem"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	btable "github.com/evertras/bubble-table/table"
)

type Component struct {
	variants         []api.Project
	configManager    *config.ConfigManager
	fileManager      *filesystem.Manager
	downloader       downloader.Downloader
	table            btable.Model
	selectedIdx      int
	downloading      bool
	progress         float64
	errorMsg         string
	infoMsg          string
	ready            bool
	atomicProgress   uint64
	currentOperation string
	selectedVariant  *api.Project
}

func New(variants []api.Project, downloader downloader.Downloader, configManager *config.ConfigManager, fileManager *filesystem.Manager) *Component {
	columns := []btable.Column{
		btable.NewColumn("desc", "Description", 32),
		btable.NewColumn("tech", "Technologies", 24),
		btable.NewColumn("diff", "Difficulty", 12),
	}
	var rows []btable.Row
	for _, v := range variants {
		rows = append(rows, btable.NewRow(map[string]interface{}{
			"desc": v.Description,
			"tech": v.Technologies,
			"diff": v.Difficulty,
		}))
	}
	table := btable.New(columns).WithRows(rows).Focused(true)
	return &Component{
		variants:      variants,
		configManager: configManager,
		fileManager:   fileManager,
		downloader:    downloader,
		table:         table,
		selectedIdx:   0,
	}
}

func (c *Component) SetDownloading(downloading bool) {
	c.downloading = downloading
	if !downloading {
		c.progress = 0
	}
}

func (c *Component) SetProgress(progress float64) {
	c.progress = progress
	atomic.StoreUint64(&c.atomicProgress, uint64(progress*100))
}

func (c *Component) GetAtomicProgress() float64 {
	return float64(atomic.LoadUint64(&c.atomicProgress)) / 100.0
}

func (c *Component) Update(msg tea.Msg) (*Component, tea.Cmd) {
	if c.downloading {
		switch msg := msg.(type) {
		case DownloadProgressMsg:
			c.SetProgress(msg.Progress)
			return c, c.progressTicker()
		case DownloadCompleteMsg:
			c.downloading = false
			c.selectedVariant = msg.Variant
			return c, nil
		case DownloadErrorMsg:
			c.downloading = false
			c.errorMsg = msg.Error
			return c, nil
		}
		return c, c.progressTicker()
	}

	c.table, _ = c.table.Update(msg)

	if m, ok := msg.(tea.KeyMsg); ok {
		switch m.String() {
		case "up", "k":
			if c.selectedIdx > 0 {
				c.selectedIdx--
			}
		case "down", "j":
			if c.selectedIdx < len(c.variants)-1 {
				c.selectedIdx++
			}
		case "enter":
			if c.selectedIdx >= 0 && c.selectedIdx < len(c.variants) {
				variant := c.variants[c.selectedIdx]
				if c.configManager != nil && c.configManager.IsProjectDownloaded(variant.ID) {
					if c.fileManager != nil {
						homeDir, err := os.UserHomeDir()
						if err == nil {
							repoName := strings.ToLower(strings.ReplaceAll(variant.Name, " ", "_"))
							projectsDir := filepath.Join(homeDir, "404skill_projects")
							entries, err := os.ReadDir(projectsDir)
							if err == nil {
								var projectDir string
								for _, entry := range entries {
									if entry.IsDir() && strings.HasPrefix(entry.Name(), repoName) {
										projectDir = filepath.Join(projectsDir, entry.Name())
										break
									}
								}
								if projectDir != "" {
									_ = c.fileManager.OpenFileExplorer(projectDir)
								}
							}
						}
					}
					c.infoMsg = "Project already downloaded. Opening project directory..."
					return c, nil
				}
				return c, c.downloadWithProgress(&variant)
			}
		case "esc", "b":
			return c, func() tea.Msg { return BackMsg{} }
		case "q", "ctrl+c":
			return c, func() tea.Msg { return QuitMsg{} }
		}
	}
	return c, nil
}

func (c *Component) downloadWithProgress(variant *api.Project) tea.Cmd {
	return tea.Batch(
		c.startDownload(variant),
		c.progressTicker(),
	)
}

func (c *Component) startDownload(variant *api.Project) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		progressCallback := func(progress float64) {
			atomic.StoreUint64(&c.atomicProgress, uint64(progress*100))
		}
		c.SetDownloading(true)
		c.currentOperation = "Cloning project..."
		err := c.downloader.DownloadProject(ctx, variant, variant.Language, progressCallback)
		if err != nil {
			return DownloadErrorMsg{Error: err.Error()}
		}
		return DownloadCompleteMsg{Variant: variant}
	}
}

func (c *Component) progressTicker() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
		return DownloadProgressMsg{Progress: c.GetAtomicProgress()}
	})
}

func (c *Component) View() string {
	if c.downloading {
		return c.renderDownloading()
	}
	view := c.renderHeader()
	view += "\n\n" + c.renderTable()
	if c.infoMsg != "" {
		view += "\n\n" + c.renderInfo()
	}
	if c.errorMsg != "" {
		view += "\n\n" + c.renderError()
	}
	return view
}

func (c *Component) renderHeader() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ffaa")).
		Bold(true).
		Underline(true).
		Padding(0, 1)
	return style.Render("Select a variant to download:")
}

func (c *Component) renderTable() string {
	return c.table.WithHighlightedRow(c.selectedIdx).View()
}

func (c *Component) renderDownloading() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ffaa")).
		Bold(true).
		Padding(0, 1)
	progress := fmt.Sprintf("Progress: %.0f%%", c.progress*100)
	return style.Render(c.currentOperation + "\n" + progress)
}

func (c *Component) renderInfo() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ffaa")).
		Bold(true)
	return style.Render(c.infoMsg)
}

func (c *Component) renderError() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ff0000")).
		Bold(true)
	return style.Render(c.errorMsg)
}

type DownloadProgressMsg struct{ Progress float64 }
type DownloadCompleteMsg struct{ Variant *api.Project }
type DownloadErrorMsg struct{ Error string }
type BackMsg struct{}
type QuitMsg struct{}
