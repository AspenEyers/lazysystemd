package app

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lazysystemd/internal/systemd"
)

const (
	leftPaneWidth = 40
	refreshInterval = 1 * time.Second
)

// ListItem represents an item in the services list (either a section header or a service)
type ListItem struct {
	IsSection bool
	SectionName string
	ServiceName string
}

// Model represents the application state
type Model struct {
	services      []*systemd.ServiceState
	items         []ListItem
	serviceMap    map[string]int // Maps service name to index in services slice
	selectedIndex int
	logLines      []string
	followMode    bool
	width         int
	height        int
	statusMessage string
	statusTimer   *time.Timer

	// Follow mode state
	followCtx    context.Context
	followCancel context.CancelFunc
	followChan   <-chan string
	followCleanup func() error
}

// NewModel creates a new model with the given list items
func NewModel(items []ListItem) *Model {
	// Build service map and extract service names
	serviceMap := make(map[string]int)
	serviceIndex := 0
	serviceNames := make([]string, 0)
	
	// Find first service index (not a section header)
	firstServiceIndex := -1
	for i, item := range items {
		if !item.IsSection {
			serviceMap[item.ServiceName] = serviceIndex
			serviceNames = append(serviceNames, item.ServiceName)
			if firstServiceIndex == -1 {
				firstServiceIndex = i
			}
			serviceIndex++
		}
	}
	
	// Default to first service if available, otherwise 0
	selectedIndex := 0
	if firstServiceIndex != -1 {
		selectedIndex = firstServiceIndex
	}
	
	return &Model{
		items:         items,
		serviceMap:    serviceMap,
		selectedIndex: selectedIndex,
		logLines:      []string{},
		followMode:    false,
		statusMessage: "Ready",
	}
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.refreshServices(),
		m.loadLogs(),
		m.tick(),
	)
}

// tick returns a command that sends a tick message every refresh interval
func (m *Model) tick() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type tickMsg time.Time

// refreshServicesCmd returns a command that refreshes all service states
type refreshServicesMsg struct {
	services []*systemd.ServiceState
	err     error
}

func (m *Model) refreshServices() tea.Cmd {
	return func() tea.Msg {
		// Count actual services (not section headers)
		serviceCount := 0
		for _, item := range m.items {
			if !item.IsSection {
				serviceCount++
			}
		}
		
		services := make([]*systemd.ServiceState, serviceCount)
		serviceIndex := 0
		for _, item := range m.items {
			if !item.IsSection {
				state, err := systemd.GetServiceState(item.ServiceName)
				if err != nil {
					state = &systemd.ServiceState{
						Name:      item.ServiceName,
						LastError: err.Error(),
					}
				}
				services[serviceIndex] = state
				// Update service map
				m.serviceMap[item.ServiceName] = serviceIndex
				serviceIndex++
			}
		}
		return refreshServicesMsg{services: services}
	}
}

// loadLogsCmd returns a command that loads logs for the selected service
type loadLogsMsg struct {
	logs []string
	err  error
}

func (m *Model) loadLogs() tea.Cmd {
	if len(m.items) == 0 || m.selectedIndex >= len(m.items) {
		return nil
	}

	// Skip section headers
	item := m.items[m.selectedIndex]
	if item.IsSection {
		return nil
	}

	serviceName := item.ServiceName
	return func() tea.Msg {
		logs, err := systemd.GetRecentLogs(serviceName, 200)
		if err != nil {
			return loadLogsMsg{err: err}
		}
		return loadLogsMsg{logs: logs}
	}
}

// startFollowMode starts following logs for the selected service
func (m *Model) startFollowMode() tea.Cmd {
	m.stopFollowMode()

	if len(m.items) == 0 || m.selectedIndex >= len(m.items) {
		return nil
	}

	// Skip section headers
	item := m.items[m.selectedIndex]
	if item.IsSection {
		return nil
	}

	serviceName := item.ServiceName
	m.followCtx, m.followCancel = context.WithCancel(context.Background())

	return func() tea.Msg {
		logChan, cleanup, err := systemd.FollowLogs(m.followCtx, serviceName)
		if err != nil {
			return followErrorMsg{err: err}
		}
		// Store cleanup in a way that's accessible
		return followStartedMsg{logChan: logChan, cleanup: cleanup}
	}
}

type followStartedMsg struct {
	logChan <-chan string
	cleanup func() error
}

type followErrorMsg struct {
	err error
}

type followLogMsg struct {
	line string
}

// stopFollowMode stops following logs
func (m *Model) stopFollowMode() {
	if m.followCancel != nil {
		m.followCancel()
		m.followCancel = nil
	}
	if m.followCleanup != nil {
		m.followCleanup()
		m.followCleanup = nil
	}
	m.followChan = nil
}

// actionMsg represents the result of a service action
type actionMsg struct {
	action string
	err    error
}

func (m *Model) performAction(action string) tea.Cmd {
	if len(m.items) == 0 || m.selectedIndex >= len(m.items) {
		return nil
	}

	// Skip section headers
	item := m.items[m.selectedIndex]
	if item.IsSection {
		return nil
	}

	serviceName := item.ServiceName
	return func() tea.Msg {
		var err error
		switch action {
		case "start":
			err = systemd.StartService(serviceName)
		case "stop":
			err = systemd.StopService(serviceName)
		case "restart":
			err = systemd.RestartService(serviceName)
		case "reload":
			err = systemd.ReloadService(serviceName)
		default:
			err = fmt.Errorf("unknown action: %s", action)
		}
		return actionMsg{action: action, err: err}
	}
}

// setStatus sets a status message that clears after 3 seconds
func (m *Model) setStatus(msg string) tea.Cmd {
	m.statusMessage = msg
	if m.statusTimer != nil {
		m.statusTimer.Stop()
	}
	m.statusTimer = time.NewTimer(3 * time.Second)
	return func() tea.Msg {
		<-m.statusTimer.C
		return statusTimeoutMsg{}
	}
}

type statusTimeoutMsg struct{}

// KeyMap defines the keybindings
type KeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Top     key.Binding
	Bottom  key.Binding
	Start   key.Binding
	Stop    key.Binding
	Restart key.Binding
	Reload  key.Binding
	Follow  key.Binding
	Refresh key.Binding
	Quit    key.Binding
}

var DefaultKeyMap = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Top: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "top"),
	),
	Bottom: key.NewBinding(
		key.WithKeys("G"),
		key.WithHelp("G", "bottom"),
	),
	Start: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "start"),
	),
	Stop: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "stop"),
	),
	Restart: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "restart"),
	),
	Reload: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "reload"),
	),
	Follow: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "follow"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("R"),
		key.WithHelp("R", "refresh"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q/ctrl+c", "quit"),
	),
}
