package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds CLI configuration with multi-layer precedence:
// flags > env > local > global > defaults
type Config struct {
	BaseURL   string `json:"base_url,omitempty"`
	Email     string `json:"email,omitempty"`
	ProjectID int    `json:"project_id,omitempty"`
	Format    string `json:"format,omitempty"`
	DevMode   bool   `json:"-"` // Set via --dev flag, not persisted
}

const (
	DefaultBaseURL = "https://api.freelo.io/v1"
)

// GlobalConfigDir returns ~/.config/freelo/
func GlobalConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "freelo")
}

// GlobalConfigPath returns ~/.config/freelo/config.json
func GlobalConfigPath() string {
	return filepath.Join(GlobalConfigDir(), "config.json")
}

// CredentialsFilename returns the credentials filename based on environment.
func CredentialsFilename(dev bool) string {
	if dev {
		return "credentials-dev.json"
	}
	return "credentials.json"
}

// Load reads config from global config file, merges with defaults.
func Load(dev bool) *Config {
	cfg := &Config{
		BaseURL: DefaultBaseURL,
		DevMode: dev,
	}

	// Dev mode: read URL from FREELO_DEV_URL env var
	if dev {
		devURL := os.Getenv("FREELO_DEV_URL")
		if devURL == "" {
			fmt.Fprintf(os.Stderr, "Error: --dev requires FREELO_DEV_URL environment variable\n")
			fmt.Fprintf(os.Stderr, "Set it with: export FREELO_DEV_URL=https://your-dev-api.example.com/v1\n")
			os.Exit(1)
		}
		cfg.BaseURL = devURL
	}

	// Load global config
	data, err := os.ReadFile(GlobalConfigPath())
	if err == nil {
		_ = json.Unmarshal(data, cfg)
	}

	// In dev mode, always force dev URL (global config cannot override)
	if dev {
		cfg.BaseURL = os.Getenv("FREELO_DEV_URL")
		cfg.DevMode = true
	}

	// Load local config (.freelo/config.json)
	// SECURITY: Local configs CANNOT override base_url (prevents credential theft
	// via malicious repos containing .freelo/config.json with a rogue base_url)
	data, err = os.ReadFile(".freelo/config.json")
	if err == nil {
		local := &Config{}
		if json.Unmarshal(data, local) == nil {
			savedBaseURL := cfg.BaseURL
			mergeConfig(cfg, local)
			cfg.BaseURL = savedBaseURL // Never allow local override of base_url
		}
	}

	// Env overrides (only when not in --dev mode)
	if !dev {
		if v := os.Getenv("FREELO_BASE_URL"); v != "" {
			cfg.BaseURL = v
		}
	}
	if v := os.Getenv("FREELO_EMAIL"); v != "" {
		cfg.Email = v
	}

	// Ensure base URL has no trailing slash
	if len(cfg.BaseURL) > 0 && cfg.BaseURL[len(cfg.BaseURL)-1] == '/' {
		cfg.BaseURL = cfg.BaseURL[:len(cfg.BaseURL)-1]
	}

	// SECURITY: Enforce HTTPS to prevent credential leakage over plain HTTP
	if cfg.BaseURL != "" && !strings.HasPrefix(cfg.BaseURL, "https://") {
		fmt.Fprintf(os.Stderr, "Warning: base_url must use HTTPS. Resetting to default.\n")
		cfg.BaseURL = DefaultBaseURL
	}

	return cfg
}

// Save writes config to the global config file.
func (c *Config) Save() error {
	dir := GlobalConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(GlobalConfigPath(), data, 0600)
}

func mergeConfig(base, overlay *Config) {
	if overlay.BaseURL != "" {
		base.BaseURL = overlay.BaseURL
	}
	if overlay.Email != "" {
		base.Email = overlay.Email
	}
	if overlay.ProjectID != 0 {
		base.ProjectID = overlay.ProjectID
	}
	if overlay.Format != "" {
		base.Format = overlay.Format
	}
}
