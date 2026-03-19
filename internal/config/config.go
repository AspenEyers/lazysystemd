package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Services []string
	Sections map[string][]string
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
		return &Config{Services: []string{}, Sections: make(map[string][]string)}, nil
	}

	// Parse YAML into a map to handle arbitrary section keys
	var rawConfig map[string]interface{}
	if err := yaml.Unmarshal(data, &rawConfig); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	config := &Config{
		Services: []string{},
		Sections: make(map[string][]string),
	}

	// Extract services and sections
	for key, value := range rawConfig {
		if key == "services" {
			// Handle services - can be either a list or a map of groups
			if servicesList, ok := value.([]interface{}); ok {
				// Backward compatibility: flat list of services
				for _, svc := range servicesList {
					if svcStr, ok := svc.(string); ok {
						config.Services = append(config.Services, svcStr)
					}
				}
			} else if servicesMap, ok := value.(map[string]interface{}); ok {
				// New format: services is a map of group names to service lists
				for groupName, groupValue := range servicesMap {
					if groupList, ok := groupValue.([]interface{}); ok {
						services := make([]string, 0, len(groupList))
						for _, svc := range groupList {
							if svcStr, ok := svc.(string); ok {
								services = append(services, svcStr)
							}
						}
						if len(services) > 0 {
							config.Sections[groupName] = services
						}
					}
				}
			}
		}
	}

	// Check if YAML is empty (no services configured)
	if len(config.Services) == 0 && len(config.Sections) == 0 {
		fmt.Fprintf(os.Stderr, "reading yaml from %s and is empty\n", path)
		return &Config{Services: []string{}, Sections: make(map[string][]string)}, nil
	}

	return config, nil
}
