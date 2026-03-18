package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Services []string `yaml:"services"`
}

// Load reads and parses the configuration file
// If the file doesn't exist, it creates an empty file.
// If the file is empty, it returns a config with empty services and prints a message.
func Load(path string) (*Config, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create empty file if it doesn't exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		file, createErr := os.Create(path)
		if createErr != nil {
			return nil, fmt.Errorf("failed to create config file: %w", createErr)
		}
		file.Close()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Check if file is empty
	if len(data) == 0 {
		fmt.Fprintf(os.Stderr, "reading yaml from %s and is empty\n", path)
		return &Config{Services: []string{}}, nil
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Check if YAML is empty (no services configured)
	if len(config.Services) == 0 {
		fmt.Fprintf(os.Stderr, "reading yaml from %s and is empty\n", path)
		return &Config{Services: []string{}}, nil
	}

	return &config, nil
}
