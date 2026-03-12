package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	configDirName  = "plane-cli"
	configFileName = "config.json"
)

// Config holds non-secret configuration persisted to disk.
type Config struct {
	ActiveProfile string             `json:"active_profile"`
	Profiles      map[string]Profile `json:"profiles"`
}

// Profile holds per-profile configuration.
type Profile struct {
	APIURL    string `json:"api_url"`
	Workspace string `json:"workspace,omitempty"`
	DocsURL   string `json:"docs_url,omitempty"`
}

// ConfigPath returns the path to the config file, respecting XDG_CONFIG_HOME.
func ConfigPath() (string, error) {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, configDirName, configFileName), nil
}

// LoadConfig reads the config file from disk. Returns a default Config if the file doesn't exist.
func LoadConfig() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{
				ActiveProfile: "default",
				Profiles:      map[string]Profile{},
			}, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	if cfg.ActiveProfile == "" {
		cfg.ActiveProfile = "default"
	}
	return &cfg, nil
}

// Save writes the config to disk with 0600 permissions.
func (c *Config) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}

// ActiveProfileConfig returns the Profile for the active profile.
func (c *Config) ActiveProfileConfig() Profile {
	if p, ok := c.Profiles[c.ActiveProfile]; ok {
		return p
	}
	return Profile{}
}

// SetProfile sets or updates a named profile.
func (c *Config) SetProfile(name string, profile Profile) {
	c.Profiles[name] = profile
}

// DeleteProfile removes a named profile.
func (c *Config) DeleteProfile(name string) {
	delete(c.Profiles, name)
}
