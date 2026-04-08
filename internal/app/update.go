package app

import (
	"fmt"

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

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case tickMsg:
		return m, tea.Batch(m.refreshServices(), m.getCPUSample(), m.tick())

	case refreshServicesMsg:
		if msg.err == nil {
			m.services = msg.services
			if msg.serviceMap != nil {
				m.serviceMap = msg.serviceMap
			}
		}
		return m, nil

	case cpuSampleMsg:
		if msg.err == nil {
			// Compute CPU usage from deltas.
			if m.cpuPrevTotal != 0 && msg.total > m.cpuPrevTotal {
				dTotal := msg.total - m.cpuPrevTotal
				dIdle := uint64(0)
				if msg.idle > m.cpuPrevIdle {
					dIdle = msg.idle - m.cpuPrevIdle
				}
				if dTotal > 0 {
					usage := (float64(dTotal-dIdle) / float64(dTotal)) * 100.0
					if usage < 0 {
						usage = 0
					}
					if usage > 100 {
						usage = 100
					}
					m.cpuUsage = usage
				}
			}
			m.cpuPrevTotal = msg.total
			m.cpuPrevIdle = msg.idle
		}
		return m, nil

	case loadLogsMsg:
		if msg.err == nil {
			m.logLines = msg.logs
		} else {
			m.logLines = []string{fmt.Sprintf("Error loading logs: %v", msg.err)}
		}
		return m, nil

	case loadCatMsg:
		if msg.err == nil {
			m.catLines = msg.catLines
		} else {
			m.catLines = []string{fmt.Sprintf("Error loading unit file: %v", msg.err)}
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
			// Human-readable success messaging (avoid naive " + ed" suffixing).
			successVerb := func(action string) string {
				switch action {
				case "start":
					return "started"
				case "stop":
					return "stopped"
				case "restart":
					return "restarted"
				case "reload":
					return "reloaded"
				case "enable":
					return "enabled"
				case "disable":
					return "disabled"
				default:
					return action + "ed"
				}
			}(msg.action)

			// Keep messaging simple and deterministic; don't rely on selection being the same
			// if actions are triggered while the view updates.
			if m.selectedIndex < len(m.items) && !m.items[m.selectedIndex].IsSection {
				m.setStatus(fmt.Sprintf("Successfully %s %s", successVerb, m.items[m.selectedIndex].ServiceName))
			} else {
				m.setStatus(fmt.Sprintf("Successfully %s", successVerb))
			}
		}

		// Refresh services after action, and refresh logs so the right pane matches the
		// currently selected service immediately.
		if m.followMode {
			return m, m.refreshServices()
		}
		return m, tea.Batch(m.refreshServices(), m.loadRightPane())

	case daemonReloadMsg:
		if msg.err != nil {
			m.setStatus(fmt.Sprintf("Failed to daemon-reload: %v", msg.err))
		} else {
			m.setStatus("Successfully ran daemon-reload")
		}
		if m.followMode {
			return m, m.refreshServices()
		}
		return m, tea.Batch(m.refreshServices(), m.loadRightPane())

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
		case <-m.followCtx.Done():
			return followStoppedMsg{}
		}
	}
}

type followStoppedMsg struct{}

// handleMouse maps clicks and wheel to focus and list selection (coordinates match lipgloss layout in View).
func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.helpMode || m.width == 0 || m.height == 0 {
		return m, nil
	}

	ch := m.mainContentHeight()
	// Row 0: header; rows 1..ch: split panes; then footer.
	if msg.Y < 1 || msg.Y > ch {
		return m, nil
	}

	leftPane := msg.X < leftPaneWidth

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if leftPane {
			return m, m.selectItemIndex(m.prevServiceIndex(m.selectedIndex))
		}
		if !m.followMode {
			m.focus = focusRight
			m.scrollRightPane(-1)
		}
		return m, nil

	case tea.MouseButtonWheelDown:
		if leftPane {
			return m, m.selectItemIndex(m.nextServiceIndex(m.selectedIndex))
		}
		if !m.followMode {
			m.focus = focusRight
			m.scrollRightPane(1)
		}
		return m, nil

	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
		if leftPane {
			innerHeight, startIdx, endIdx := m.listScrollWindow(ch)
			pr := msg.Y - 1 // row within full-height content strip
			if pr < 1 || pr > innerHeight {
				return m, nil
			}
			v := pr - 1
			visible := endIdx - startIdx
			if v >= visible {
				return m, nil
			}
			idx := startIdx + v
			if idx < 0 || idx >= len(m.items) {
				return m, nil
			}
			m.focus = focusLeft
			if m.items[idx].IsSection {
				return m, nil
			}
			return m, m.selectItemIndex(idx)
		}
		m.focus = focusRight
		if m.rightMode == rightPaneLogs && m.logTop < 0 {
			m.logTop = len(m.logLines)
		}
		return m, nil

	default:
		return m, nil
	}
}

// handleKey processes keyboard input
func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	keyMap := DefaultKeyMap
	var cmd tea.Cmd

	// If help is open, only allow closing it.
	if m.helpMode {
		switch {
		case key.Matches(msg, keyMap.Help), key.Matches(msg, keyMap.Escape):
			m.helpMode = false
			return m, nil
		default:
			return m, nil
		}
	}

	switch {
	case key.Matches(msg, keyMap.Quit):
		m.stopFollowMode()
		return m, tea.Quit

	case key.Matches(msg, keyMap.Help):
		m.helpMode = true
		return m, nil

	case key.Matches(msg, keyMap.Theme):
		m.invertTheme = !m.invertTheme
		return m, nil

	case key.Matches(msg, keyMap.Left):
		m.focus = focusLeft
		return m, nil

	case key.Matches(msg, keyMap.Right):
		m.focus = focusRight
		// Initialize logTop to tail when focusing logs for the first time.
		if m.rightMode == rightPaneLogs && m.logTop < 0 {
			m.logTop = len(m.logLines)
		}
		return m, nil

	case key.Matches(msg, keyMap.Escape):
		// Esc closes `cat` and returns to logs.
		if m.rightMode == rightPaneCat && !m.followMode {
			m.rightMode = rightPaneLogs
			m.focus = focusLeft
			m.logTop = -1
			return m, m.loadLogs()
		}
		return m, nil

	case key.Matches(msg, keyMap.Up):
		// If the right pane is focused, scroll instead of changing selection.
		if m.focus == focusRight && !m.followMode {
			switch m.rightMode {
			case rightPaneCat:
				if m.catTop > 0 {
					m.catTop--
				}
				return m, nil
			case rightPaneLogs:
				if m.logTop < 0 {
					m.logTop = len(m.logLines)
				}
				if m.logTop > 0 {
					m.logTop--
				}
				return m, nil
			}
		}

		// Move up, skipping section headers
		oldIndex := m.selectedIndex
		wasFollowing := m.followMode
		for i := m.selectedIndex - 1; i >= 0; i-- {
			if !m.items[i].IsSection {
				m.selectedIndex = i
				m.stopFollowMode()
				m.focus = focusLeft
				m.logTop = -1
				m.catTop = 0
				if wasFollowing {
					// Keep follow enabled but restart it for the newly selected service.
					m.followMode = true
					cmd = tea.Batch(m.loadLogs(), m.startFollowMode())
				} else {
					m.followMode = false
					cmd = m.loadRightPane()
				}
				break
			}
		}
		if m.selectedIndex == oldIndex {
			cmd = nil
		}

	case key.Matches(msg, keyMap.Down):
		// If the right pane is focused, scroll instead of changing selection.
		if m.focus == focusRight && !m.followMode {
			switch m.rightMode {
			case rightPaneCat:
				if m.catTop < max(0, len(m.catLines)-1) {
					m.catTop++
				}
				return m, nil
			case rightPaneLogs:
				if m.logTop < 0 {
					m.logTop = len(m.logLines)
				}
				if m.logTop < len(m.logLines) {
					m.logTop++
				}
				return m, nil
			}
		}

		// Move down, skipping section headers
		oldIndex := m.selectedIndex
		wasFollowing := m.followMode
		for i := m.selectedIndex + 1; i < len(m.items); i++ {
			if !m.items[i].IsSection {
				m.selectedIndex = i
				m.stopFollowMode()
				m.focus = focusLeft
				m.logTop = -1
				m.catTop = 0
				if wasFollowing {
					m.followMode = true
					cmd = tea.Batch(m.loadLogs(), m.startFollowMode())
				} else {
					m.followMode = false
					cmd = m.loadRightPane()
				}
				break
			}
		}
		if m.selectedIndex == oldIndex {
			cmd = nil
		}

	case key.Matches(msg, keyMap.Top):
		wasFollowing := m.followMode
		// Jump to first service (not section header)
		for i := 0; i < len(m.items); i++ {
			if !m.items[i].IsSection {
				if m.selectedIndex != i {
					m.selectedIndex = i
					m.stopFollowMode()
					if wasFollowing {
						m.followMode = true
						cmd = tea.Batch(m.loadLogs(), m.startFollowMode())
					} else {
						m.followMode = false
						cmd = m.loadRightPane()
					}
				}
				break
			}
		}

	case key.Matches(msg, keyMap.Bottom):
		wasFollowing := m.followMode
		// Jump to last service (not section header)
		for i := len(m.items) - 1; i >= 0; i-- {
			if !m.items[i].IsSection {
				if m.selectedIndex != i {
					m.selectedIndex = i
					m.stopFollowMode()
					if wasFollowing {
						m.followMode = true
						cmd = tea.Batch(m.loadLogs(), m.startFollowMode())
					} else {
						m.followMode = false
						cmd = m.loadRightPane()
					}
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

	case key.Matches(msg, keyMap.Enable):
		cmd = m.performAction("enable")

	case key.Matches(msg, keyMap.Disable):
		cmd = m.performAction("disable")

	case key.Matches(msg, keyMap.DaemonReload):
		cmd = m.performDaemonReload()

	case key.Matches(msg, keyMap.Follow):
		m.followMode = !m.followMode
		m.rightMode = rightPaneLogs
		m.focus = focusLeft
		m.logTop = -1
		if m.followMode {
			// Load recent logs immediately, then start following new lines.
			cmd = tea.Batch(m.loadLogs(), m.startFollowMode())
		} else {
			m.stopFollowMode()
			// Stop-follow switches back to "recent logs" view.
			cmd = m.loadLogs()
		}

	case key.Matches(msg, keyMap.Refresh):
		cmd = tea.Batch(m.refreshServices(), m.loadRightPane())

	case key.Matches(msg, keyMap.Enter):
		// Show `systemctl cat` for the selected service (ignore section headers).
		if m.selectedIndex < len(m.items) && !m.items[m.selectedIndex].IsSection {
			m.stopFollowMode()
			m.followMode = false
			m.rightMode = rightPaneCat
			m.focus = focusRight
			m.catTop = 0
			cmd = m.loadCat()
		}
	}

	return m, cmd
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
