package controller

import (
	"404skill-cli/tui/styles"

	"github.com/charmbracelet/lipgloss"
)

// View rendering functions

func (c *Controller) renderQuitting() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ff0000")).
		Bold(true).
		Render("Goodbye!") + "\n"
}

func (c *Controller) renderRefreshingToken() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ffaa")).
		Bold(true).
		Underline(true).
		Padding(0, 1).
		Render("\nRefreshing session... Please wait.")
}

func (c *Controller) renderMainMenu() string {
	view := styles.GetASCIIArt(styles.VersionInfo{
		CurrentVersion:  c.versionInfo.CurrentVersion,
		LatestVersion:   c.versionInfo.LatestVersion,
		UpdateAvailable: c.versionInfo.UpdateAvailable,
		CheckError:      c.versionInfo.CheckError,
	}) + "\n"
	view += c.mainMenu.View()
	view += "\n" + c.footer.View(c.footerBindings.Navigation()...)
	return view
}

func (c *Controller) renderLogin() string {
	return c.loginComponent.View()
}

func (c *Controller) renderProjectNameMenu() string {
	if c.loading {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00ffaa")).
			Bold(true).
			Underline(true).
			Padding(0, 1).
			Render("\nLoading projects...")
	}

	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ffaa")).
		Bold(true).
		Underline(true).
		Padding(0, 1).
		Render("Select a project:")

	return header + "\n" + c.projectNameMenu.View() + "\n" + c.footer.View(c.footerBindings.NavigationWithBack()...)
}

func (c *Controller) renderProjectVariantMenu() string {
	if c.variantComponent != nil {
		componentView := c.variantComponent.View()
		// Don't show footer when downloading (component handles its own controls)
		if c.variantComponent.IsDownloading() {
			return componentView
		}
		return componentView + "\n" + c.footer.View(c.footerBindings.NavigationWithBack()...)
	}
	return "No variants available."
}

func (c *Controller) renderTestProject() string {
	if c.loading {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00ffaa")).
			Bold(true).
			Underline(true).
			Padding(0, 1).
			Render("\nLoading projects...")
	}
	return c.testComponent.View()
}

func (c *Controller) renderTestProjectNameMenu() string {
	if c.loading {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00ffaa")).
			Bold(true).
			Underline(true).
			Padding(0, 1).
			Render("\nLoading projects...")
	}

	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ffaa")).
		Bold(true).
		Underline(true).
		Padding(0, 1).
		Render("Select a project to test:")

	return header + "\n" + c.testProjectNameMenu.View() + "\n" + c.footer.View(c.footerBindings.NavigationWithBack()...)
}

func (c *Controller) renderTestProjectVariantMenu() string {
	if c.testVariantComponent != nil {
		componentView := c.testVariantComponent.View()
		// Don't show footer when testing (component handles its own controls)
		if c.testVariantComponent.IsTesting() {
			return componentView
		}
		return componentView + "\n" + c.footer.View(c.footerBindings.NavigationWithBack()...)
	}
	return "No variants available."
}
