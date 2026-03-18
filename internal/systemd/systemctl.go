package systemd

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

// ServiceState represents the state of a systemd service
type ServiceState struct {
	Name          string
	Description   string
	LoadState     string
	ActiveState   string
	SubState      string
	UnitFileState string
	MainPID       string
	LastError     string
}

// GetServiceState retrieves the state of a systemd service unit
func GetServiceState(unitName string) (*ServiceState, error) {
	cmd := exec.Command("systemctl", "show", unitName,
		"--no-pager",
		"--property=Id,Description,LoadState,ActiveState,SubState,UnitFileState,MainPID")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("systemctl show failed: %w", err)
	}

	state := &ServiceState{Name: unitName}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		switch key {
		case "Id":
			state.Name = value
		case "Description":
			state.Description = value
		case "LoadState":
			state.LoadState = value
		case "ActiveState":
			state.ActiveState = value
		case "SubState":
			state.SubState = value
		case "UnitFileState":
			state.UnitFileState = value
		case "MainPID":
			state.MainPID = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse systemctl output: %w", err)
	}

	return state, nil
}

// GetStateIndicator returns a compact state indicator for display
func (s *ServiceState) GetStateIndicator() string {
	switch s.ActiveState {
	case "active":
		if s.SubState == "running" {
			return "●"
		}
		return "○"
	case "inactive":
		return "○"
	case "failed":
		return "✗"
	case "activating":
		return "→"
	case "deactivating":
		return "←"
	default:
		return "?"
	}
}


// StartService starts a systemd service
func StartService(unitName string) error {
	cmd := exec.Command("systemctl", "start", unitName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}
	return nil
}

// StopService stops a systemd service
func StopService(unitName string) error {
	cmd := exec.Command("systemctl", "stop", unitName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}
	return nil
}

// RestartService restarts a systemd service
func RestartService(unitName string) error {
	cmd := exec.Command("systemctl", "restart", unitName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restart service: %w", err)
	}
	return nil
}

// ReloadService reloads a systemd service
func ReloadService(unitName string) error {
	cmd := exec.Command("systemctl", "reload", unitName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload service: %w", err)
	}
	return nil
}
