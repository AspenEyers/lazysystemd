package systemd

import (
	"bufio"
	"fmt"
	"os"
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

// getScopeFlag returns "--system" if running as root, "--user" otherwise
func getScopeFlag() string {
	if os.Geteuid() == 0 {
		return "--system"
	}
	return "--user"
}

// GetServiceState retrieves the state of a systemd service unit
func GetServiceState(unitName string) (*ServiceState, error) {
	cmd := exec.Command("systemctl", getScopeFlag(), "show", unitName,
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

// GetServiceCat runs `systemctl cat <unit>` and returns the unit file contents.
func GetServiceCat(unitName string) ([]string, error) {
	// `systemctl cat` prints the unit file and any drop-ins.
	cmd := exec.Command("systemctl", getScopeFlag(), "--no-pager", "cat", unitName)
	output, err := cmd.Output()
	if err != nil {
		// Include stdout/stderr context via error string (systemctl tends to print useful info).
		return nil, fmt.Errorf("systemctl cat failed: %w", err)
	}

	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse systemctl cat output: %w", err)
	}
	return lines, nil
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

// GetEnabledIndicator returns a compact enabled/disabled indicator for display.
func (s *ServiceState) GetEnabledIndicator() string {
	switch s.UnitFileState {
	case "enabled":
		return "E"
	case "disabled":
		return "D"
	case "static":
		return "S"
	case "masked":
		return "M"
	case "indirect":
		return "I"
	case "":
		return "?"
	default:
		return "?"
	}
}


// StartService starts a systemd service
func StartService(unitName string) error {
	cmd := exec.Command("systemctl", getScopeFlag(), "start", unitName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}
	return nil
}

// StopService stops a systemd service
func StopService(unitName string) error {
	cmd := exec.Command("systemctl", getScopeFlag(), "stop", unitName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}
	return nil
}

// RestartService restarts a systemd service
func RestartService(unitName string) error {
	cmd := exec.Command("systemctl", getScopeFlag(), "restart", unitName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restart service: %w", err)
	}
	return nil
}

// ReloadService reloads a systemd service
func ReloadService(unitName string) error {
	cmd := exec.Command("systemctl", getScopeFlag(), "reload", unitName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload service: %w", err)
	}
	return nil
}

// EnableService enables a systemd service unit.
func EnableService(unitName string) error {
	cmd := exec.Command("systemctl", getScopeFlag(), "enable", unitName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}
	return nil
}

// DisableService disables a systemd service unit.
func DisableService(unitName string) error {
	cmd := exec.Command("systemctl", getScopeFlag(), "disable", unitName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to disable service: %w", err)
	}
	return nil
}

// DaemonReload runs "systemctl daemon-reload".
func DaemonReload() error {
	cmd := exec.Command("systemctl", getScopeFlag(), "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to daemon-reload: %w", err)
	}
	return nil
}
