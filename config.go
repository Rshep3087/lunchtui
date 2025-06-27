package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// Config represents the application configuration structure.
type Config struct {
	// Debug enables debug logging
	Debug bool `toml:"debug"`
	// Token is the Lunch Money API token
	Token string `toml:"token"`
	// DebitsAsNegative shows debits as negative numbers
	DebitsAsNegative bool `toml:"debits_as_negative"`

	configPathUsed string // Path to the configuration file used
}

// getConfigFilePaths returns the list of possible configuration file paths
// in order of precedence (first found wins).
func getConfigFilePaths() []string {
	var paths []string

	// Current directory (highest precedence)
	paths = append(paths, "lunchtui.toml")

	// User config directory
	if configDir, err := os.UserConfigDir(); err == nil {
		paths = append(paths, filepath.Join(configDir, "lunchtui", "config.toml"))
	}

	// User home directory
	if homeDir, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(homeDir, ".lunchtui.toml"))
		paths = append(paths, filepath.Join(homeDir, ".config", "lunchtui", "config.toml"))
	}

	// System-wide config directory (lowest precedence)
	paths = append(paths, "/etc/lunchtui/config.toml")

	return paths
}

// findConfigFile searches for a configuration file in the standard locations.
// Returns the path to the first existing config file, or empty string if none found.
func findConfigFile() string {
	for _, path := range getConfigFilePaths() {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// loadConfigFromFile loads configuration from a TOML file.
func loadConfigFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var config Config
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse TOML config file %s: %w", path, err)
	}

	return &config, nil
}

// loadConfig loads configuration from file if available, otherwise returns default config.
func loadConfig() (*Config, string, error) {
	configPath := findConfigFile()
	if configPath == "" {
		// No config file found, return default configuration
		return &Config{}, "", nil
	}

	config, err := loadConfigFromFile(configPath)
	if err != nil {
		return nil, configPath, err
	}

	return config, configPath, nil
}
