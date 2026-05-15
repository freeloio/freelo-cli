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

// Load reads config from global config file, merges with defaults.
//
// Returns an error rather than calling os.Exit so the caller controls
// when to bail out — important for `--help` and `version` flows which
// should keep working even when `--dev` is missing FREELO_DEV_URL.
func Load(dev bool) (*Config, error) {
	cfg := &Config{
		BaseURL: DefaultBaseURL,
		DevMode: dev,
	}

	// Dev mode: read URL from FREELO_DEV_URL env var. We collect the
	// dev URL but only enforce it later — Load shouldn't fatal on
	// --help.
	var devURL string
	if dev {
		devURL = os.Getenv("FREELO_DEV_URL")
		if devURL == "" {
			return nil, fmt.Errorf("--dev requires FREELO_DEV_URL environment variable (set it with: export FREELO_DEV_URL=https://your-dev-api.example.com/v1)")
		}
		cfg.BaseURL = devURL
	}

	// Load global config. We tolerate ENOENT (no config yet) but surface
	// malformed JSON as a stderr warning — silently overwriting the
	// user's config because we couldn't read it would be worse than
	// letting them see the message.
	if data, err := os.ReadFile(GlobalConfigPath()); err == nil {
		if jerr := json.Unmarshal(data, cfg); jerr != nil {
			fmt.Fprintf(os.Stderr, "warning: %s is not valid JSON (%v); using defaults\n", GlobalConfigPath(), jerr)
		}
	}

	// In dev mode, always force dev URL (global config cannot override)
	if dev {
		cfg.BaseURL = devURL
		cfg.DevMode = true
	}

	// Load local config (.freelo/config.json)
	// SECURITY: Local configs CANNOT override base_url (prevents credential theft
	// via malicious repos containing .freelo/config.json with a rogue base_url)
	if data, err := os.ReadFile(".freelo/config.json"); err == nil {
		local := &Config{}
		if jerr := json.Unmarshal(data, local); jerr != nil {
			fmt.Fprintf(os.Stderr, "warning: .freelo/config.json is not valid JSON (%v); ignoring\n", jerr)
		} else {
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
	cfg.BaseURL = strings.TrimSuffix(cfg.BaseURL, "/")

	// SECURITY: Enforce HTTPS to prevent credential leakage over plain HTTP
	if cfg.BaseURL != "" && !strings.HasPrefix(cfg.BaseURL, "https://") {
		fmt.Fprintf(os.Stderr, "warning: base_url %q does not use HTTPS; resetting to default %s\n", cfg.BaseURL, DefaultBaseURL)
		cfg.BaseURL = DefaultBaseURL
	}

	return cfg, nil
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
