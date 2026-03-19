package app

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Ensure selectedIndex doesn't point to a section header
	if m.selectedIndex < len(m.items) && m.items[m.selectedIndex].IsSection {
		// Find next service
		for i := m.selectedIndex + 1; i < len(m.items); i++ {
			if !m.items[i].IsSection {
				m.selectedIndex = i
				break
			}
		}
		// If no service found after, try before
		if m.selectedIndex < len(m.items) && m.items[m.selectedIndex].IsSection {
			for i := m.selectedIndex - 1; i >= 0; i-- {
				if !m.items[i].IsSection {
					m.selectedIndex = i
					break
				}
			}
		}
	}
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tickMsg:
		return m, tea.Batch(m.refreshServices(), m.tick())

	case refreshServicesMsg:
		if msg.err == nil {
			m.services = msg.services
		}
		return m, nil

	case loadLogsMsg:
		if msg.err == nil {
			m.logLines = msg.logs
		} else {
			m.logLines = []string{fmt.Sprintf("Error loading logs: %v", msg.err)}
		}
		return m, nil

	case followStartedMsg:
		m.followChan = msg.logChan
		m.followCleanup = msg.cleanup
		return m, m.watchFollowChan()

	case followErrorMsg:
		m.setStatus(fmt.Sprintf("Follow error: %v", msg.err))
		m.followMode = false
		return m, nil

	case followLogMsg:
		m.logLines = append(m.logLines, msg.line)
		// Keep only last 500 lines
		if len(m.logLines) > 500 {
			m.logLines = m.logLines[len(m.logLines)-500:]
		}
		return m, m.watchFollowChan()

	case followStoppedMsg:
		m.followMode = false
		m.followChan = nil
		return m, nil

	case actionMsg:
		if msg.err != nil {
			m.setStatus(fmt.Sprintf("Failed to %s: %v", msg.action, msg.err))
		} else {
			if m.selectedIndex < len(m.items) && !m.items[m.selectedIndex].IsSection {
				m.setStatus(fmt.Sprintf("Successfully %sed %s", msg.action, m.items[m.selectedIndex].ServiceName))
			}
		}
		// Refresh services after action
		return m, m.refreshServices()

	case statusTimeoutMsg:
		m.statusMessage = "Ready"
		return m, nil
	}

	return m, nil
}

// watchFollowChan watches the follow channel for new log lines
func (m *Model) watchFollowChan() tea.Cmd {
	if m.followChan == nil {
		return nil
	}

	return func() tea.Msg {
		select {
		case line, ok := <-m.followChan:
			if !ok {
				return followStoppedMsg{}
			}
			return followLogMsg{line: line}
		case <-time.After(100 * time.Millisecond):
			return nil
		}
	}
}

type followStoppedMsg struct{}

// handleKey processes keyboard input
func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	keyMap := DefaultKeyMap
	var cmd tea.Cmd

	switch {
	case key.Matches(msg, keyMap.Quit):
		m.stopFollowMode()
		return m, tea.Quit

	case key.Matches(msg, keyMap.Up):
		// Move up, skipping section headers
		for i := m.selectedIndex - 1; i >= 0; i-- {
			if !m.items[i].IsSection {
				m.selectedIndex = i
				m.stopFollowMode()
				m.followMode = false
				cmd = m.loadLogs()
				break
			}
		}

	case key.Matches(msg, keyMap.Down):
		// Move down, skipping section headers
		for i := m.selectedIndex + 1; i < len(m.items); i++ {
			if !m.items[i].IsSection {
				m.selectedIndex = i
				m.stopFollowMode()
				m.followMode = false
				cmd = m.loadLogs()
				break
			}
		}

	case key.Matches(msg, keyMap.Top):
		// Jump to first service (not section header)
		for i := 0; i < len(m.items); i++ {
			if !m.items[i].IsSection {
				if m.selectedIndex != i {
					m.selectedIndex = i
					m.stopFollowMode()
					m.followMode = false
					cmd = m.loadLogs()
				}
				break
			}
		}

	case key.Matches(msg, keyMap.Bottom):
		// Jump to last service (not section header)
		for i := len(m.items) - 1; i >= 0; i-- {
			if !m.items[i].IsSection {
				if m.selectedIndex != i {
					m.selectedIndex = i
					m.stopFollowMode()
					m.followMode = false
					cmd = m.loadLogs()
				}
				break
			}
		}

	case key.Matches(msg, keyMap.Start):
		cmd = m.performAction("start")

	case key.Matches(msg, keyMap.Stop):
		cmd = m.performAction("stop")

	case key.Matches(msg, keyMap.Restart):
		cmd = m.performAction("restart")

	case key.Matches(msg, keyMap.Reload):
		cmd = m.performAction("reload")

	case key.Matches(msg, keyMap.Follow):
		m.followMode = !m.followMode
		if m.followMode {
			cmd = tea.Batch(m.startFollowMode(), m.watchFollowChan())
		} else {
			m.stopFollowMode()
		}

	case key.Matches(msg, keyMap.Refresh):
		cmd = m.refreshServices()
	}

	return m, cmd
}
