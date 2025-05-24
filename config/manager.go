package config

// ConfigManager handles configuration operations
type ConfigManager struct{}

// NewConfigManager creates a new config manager
func NewConfigManager() *ConfigManager {
	return &ConfigManager{}
}

// IsProjectDownloaded checks if a project has been downloaded
func (c *ConfigManager) IsProjectDownloaded(projectID string) bool {
	cfg, err := ReadConfig()
	if err != nil {
		return false
	}
	return cfg.DownloadedProjects != nil && cfg.DownloadedProjects[projectID]
}

// UpdateDownloadedProject marks a project as downloaded
func (c *ConfigManager) UpdateDownloadedProject(projectID string) error {
	cfg, err := ReadConfig()
	if err != nil {
		return err
	}
	if cfg.DownloadedProjects == nil {
		cfg.DownloadedProjects = make(map[string]bool)
	}
	cfg.DownloadedProjects[projectID] = true
	return WriteConfig(cfg)
}
