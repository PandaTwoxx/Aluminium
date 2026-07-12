package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const DefaultServerURL = "https://aluminium-server.warmraisin.com"

type ServerConfig struct {
	Token string `json:"token"`
}

type Config struct {
	DefaultServer string                  `json:"default_server"`
	SearchServers []string                `json:"search_servers"`
	Servers       map[string]ServerConfig `json:"servers"`
}

// GetConfigDir returns the path to the configuration directory: ~/.aluminium/
func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".aluminium")
	return dir, nil
}

// GetConfigFilePath returns the path to ~/.aluminium/config.json
func GetConfigFilePath() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// NewDefaultConfig initializes a new Config with default values
func NewDefaultConfig() *Config {
	return &Config{
		DefaultServer: DefaultServerURL,
		SearchServers: []string{DefaultServerURL},
		Servers:       make(map[string]ServerConfig),
	}
}

// LoadConfig loads configuration from ~/.aluminium/config.json,
// creating it with defaults if it does not exist.
func LoadConfig() (*Config, error) {
	path, err := GetConfigFilePath()
	if err != nil {
		return nil, err
	}

	// Create directories if they do not exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg := NewDefaultConfig()
			if err := SaveConfig(cfg); err != nil {
				return nil, err
			}
			return cfg, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Initialize maps and slices if nil
	if cfg.Servers == nil {
		cfg.Servers = make(map[string]ServerConfig)
	}
	if len(cfg.SearchServers) == 0 {
		cfg.SearchServers = []string{DefaultServerURL}
	}
	if cfg.DefaultServer == "" {
		cfg.DefaultServer = DefaultServerURL
	}

	return &cfg, nil
}

// SaveConfig saves the configuration to ~/.aluminium/config.json
func SaveConfig(cfg *Config) error {
	path, err := GetConfigFilePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}
