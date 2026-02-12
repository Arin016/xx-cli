// Package config handles loading and persisting user configuration
// for the xx-cli tool. Configuration is stored in ~/.xx-cli/config.json.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	dirName      = ".xx-cli"
	fileName     = "config.json"
	defaultModel = "llama3.2:latest"
	envKeyModel  = "XX_MODEL"
)

// Config holds the user's configuration.
type Config struct {
	APIKey string `json:"api_key,omitempty"`
	Model  string `json:"model"`
}

// Dir returns the configuration directory path.
func Dir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, dirName)
}

func configPath() string {
	return filepath.Join(Dir(), fileName)
}

// Load reads the configuration from disk and environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		Model: defaultModel,
	}

	data, err := os.ReadFile(configPath())
	if err == nil {
		_ = json.Unmarshal(data, cfg)
	}

	if model := os.Getenv(envKeyModel); model != "" {
		cfg.Model = model
	}

	if cfg.Model == "" {
		cfg.Model = defaultModel
	}

	return cfg, nil
}

// save persists the config to disk.
func save(cfg *Config) error {
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath(), data, 0o600)
}

// SetAPIKey saves the API key to the config file.
func SetAPIKey(key string) error {
	cfg := &Config{Model: defaultModel}

	data, err := os.ReadFile(configPath())
	if err == nil {
		_ = json.Unmarshal(data, cfg)
	}

	cfg.APIKey = key
	return save(cfg)
}

// SetModel saves the model preference to the config file.
func SetModel(model string) error {
	cfg := &Config{Model: defaultModel}

	data, err := os.ReadFile(configPath())
	if err == nil {
		_ = json.Unmarshal(data, cfg)
	}

	cfg.Model = model
	return save(cfg)
}
