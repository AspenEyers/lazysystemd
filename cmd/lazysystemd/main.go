package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lazysystemd/internal/app"
	"github.com/lazysystemd/internal/config"
)

func main() {
	// Default to $HOME/.config/lazysystemd/config.yaml
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		os.Exit(1)
	}
	defaultConfig := filepath.Join(homeDir, ".config", "lazysystemd", "config.yaml")

	configPath := flag.String("config", defaultConfig, "Path to configuration file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Convert config to list items
	items := make([]app.ListItem, 0)
	
	// Add top-level services first (if any)
	if len(cfg.Services) > 0 {
		for _, service := range cfg.Services {
			items = append(items, app.ListItem{
				IsSection:   false,
				ServiceName: service,
			})
		}
	}
	
	// Add sections (iterate in a deterministic order)
	sectionNames := make([]string, 0, len(cfg.Sections))
	for name := range cfg.Sections {
		sectionNames = append(sectionNames, name)
	}
	// Sort section names for consistent ordering
	for i := 0; i < len(sectionNames)-1; i++ {
		for j := i + 1; j < len(sectionNames); j++ {
			if sectionNames[i] > sectionNames[j] {
				sectionNames[i], sectionNames[j] = sectionNames[j], sectionNames[i]
			}
		}
	}
	
	for _, sectionName := range sectionNames {
		// Add section header
		items = append(items, app.ListItem{
			IsSection:   true,
			SectionName: sectionName,
		})
		// Add services in this section
		for _, service := range cfg.Sections[sectionName] {
			items = append(items, app.ListItem{
				IsSection:   false,
				ServiceName: service,
			})
		}
	}

	model := app.NewModel(items)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
