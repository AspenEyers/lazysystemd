package app

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/lazysystemd/internal/systemd"
)

type uiStyles struct {
	title         lipgloss.Style
	selected      lipgloss.Style
	selectedCat   lipgloss.Style
	normal        lipgloss.Style
	status        lipgloss.Style
	error         lipgloss.Style
	success       lipgloss.Style
	border        lipgloss.Style
	focusedBorder lipgloss.Style
	footer        lipgloss.Style
}

func (m *Model) styles() uiStyles {
	blue := lipgloss.Color("39")

	// Default: dark theme
	if !m.invertTheme {
		return uiStyles{
			title: lipgloss.NewStyle().Bold(true).Foreground(blue),
			selected: lipgloss.NewStyle().
				Bold(true).
				Background(lipgloss.Color("62")).
				Foreground(lipgloss.Color("230")),
			selectedCat: lipgloss.NewStyle().
				Bold(true).
				Background(lipgloss.Color("62")).
				Foreground(blue),
			normal:  lipgloss.NewStyle().Foreground(lipgloss.Color("250")),
			status:  lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
			error:   lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
			success: lipgloss.NewStyle().Foreground(lipgloss.Color("46")),
			border: lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62")),
			focusedBorder: lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(blue),
			footer: lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Height(2),
		}
	}

	// Inverted: light theme
	return uiStyles{
		title: lipgloss.NewStyle().Bold(true).Foreground(blue).Background(lipgloss.Color("15")),
		selected: lipgloss.NewStyle().
			Bold(true).
			Background(blue).
			Foreground(lipgloss.Color("15")),
		selectedCat: lipgloss.NewStyle().
			Bold(true).
			Background(blue).
			Foreground(lipgloss.Color("15")),
		normal:  lipgloss.NewStyle().Foreground(lipgloss.Color("232")).Background(lipgloss.Color("15")),
		status:  lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Background(lipgloss.Color("15")),
		error:   lipgloss.NewStyle().Foreground(lipgloss.Color("160")).Background(lipgloss.Color("15")),
		success: lipgloss.NewStyle().Foreground(lipgloss.Color("28")).Background(lipgloss.Color("15")),
		border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("244")).
			Background(lipgloss.Color("15")),
		focusedBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(blue).
			Background(lipgloss.Color("15")),
		footer: lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Background(lipgloss.Color("15")).Height(2),
	}
}

// View renders the UI
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	s := m.styles()

	// When help is open, render a full-screen help panel.
	if m.helpMode {
		return m.renderHelp()
	}

	// Calculate pane heights.
	// `renderServicesList` and `renderLogs` both wrap their content in a lipgloss border,
	// so we size the panes to fit within the terminal reliably.
	headerHeight := 1
	footerHeight := 2
	// Bubble Tea / terminal UIs are commonly off-by-one in the reported height.
	// Reserve a couple extra lines to prevent any top/bottom clipping.
	availableHeight := m.height - 2
	if availableHeight < 1 {
		availableHeight = 1
	}
	contentHeight := availableHeight - headerHeight - footerHeight

	if contentHeight < 1 {
		contentHeight = 1
	}

	// Left pane: services list
	leftPane := m.renderServicesList(contentHeight)

	// Right pane: logs or systemctl cat
	rightPane := m.renderRightPane(contentHeight)

	// Combine panes
	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	// Header
	host := m.hostname
	if host == "" {
		host = "unknown"
	}
	now := time.Now()
	const clockFmt = "2006/01/02 15:04:05"
	headerText := fmt.Sprintf("lazysystemd - %s - CPU %5.1f%% - (UTC) %s | (local) %s",
		host, m.cpuUsage, now.UTC().Format(clockFmt), now.Format(clockFmt))
	header := s.title.Height(headerHeight).Render(headerText)

	// Footer
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		content,
		footer,
	)
}

func (m *Model) renderRightPane(height int) string {
	// Follow mode always displays logs.
	if m.followMode || m.rightMode == rightPaneLogs {
		return m.renderLogs(height)
	}
	return m.renderCat(height)
}

func (m *Model) renderHelp() string {
	s := m.styles()
	headerHeight := 1
	footerHeight := 2

	availableHeight := m.height - 2
	if availableHeight < 1 {
		availableHeight = 1
	}
	contentHeight := availableHeight - headerHeight - footerHeight
	if contentHeight < 1 {
		contentHeight = 1
	}

	header := s.title.Height(headerHeight).Render("lazysystemd (help)")
	footer := m.renderFooter()

	innerHeight := contentHeight - 2
	if innerHeight < 1 {
		innerHeight = 1
	}

	maxWidth := m.width - 2
	if maxWidth < 1 {
		maxWidth = 1
	}

	lines := []string{
		"Keys",
		"  ↑/k, ↓/j      Move selection",
		"  g / G         Top / bottom",
		"  s             Start",
		"  t             Stop",
		"  r             Restart",
		"  L             Reload selected service",
		"  e             Enable selected service",
		"  d             Disable selected service",
		"  l             systemctl daemon-reload",
		"  f             Toggle follow mode (live logs)",
		"  R             Refresh statuses (and right pane)",
		"  Enter         Show `systemctl cat` for selected service",
		"  p             Toggle theme (invert colors)",
		"  q/ctrl+c      Quit",
		"",
		"Icons",
		"  ● active (running)   ○ inactive",
		"  ✗ failed              → activating",
		"  ← deactivating       ? unknown",
		"  Enabled overlay:",
		"    ◎ enabled+inactive (double ring hollow)",
		"    ◉ enabled+running  (double ring bullseye)",
		"    disabled: no outer circle (only ○/●)",
		"",
		"Close help: h or esc",
	}

	// Truncate long lines to fit.
	for i := range lines {
		if utf8.RuneCountInString(lines[i]) > maxWidth {
			r := []rune(lines[i])
			lines[i] = string(r[:maxWidth]) + "…"
		}
	}

	// Pad/truncate to fit inside border.
	for len(lines) < innerHeight {
		lines = append(lines, "")
	}
	if len(lines) > innerHeight {
		lines = lines[:innerHeight]
	}

	content := strings.Join(lines, "\n")
	helpBox := s.border.Width(m.width).Height(contentHeight).Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, header, helpBox, footer)
}

// renderServicesList renders the left pane with the services list
func (m *Model) renderServicesList(height int) string {
	s := m.styles()
	var lines []string
	innerHeight := height - 2 // lipgloss border consumes top+bottom
	if innerHeight < 1 {
		innerHeight = 1
	}

	// Scroll window so the selected row stays visible.
	startIdx := 0
	if len(m.items) > innerHeight {
		startIdx = m.selectedIndex - (innerHeight / 2)
		if startIdx < 0 {
			startIdx = 0
		}
		maxStart := len(m.items) - innerHeight
		if startIdx > maxStart {
			startIdx = maxStart
		}
	}
	endIdx := startIdx + innerHeight
	if endIdx > len(m.items) {
		endIdx = len(m.items)
	}

	for i := startIdx; i < endIdx; i++ {
		item := m.items[i]
		var line string
		
		if item.IsSection {
			// Render section header
			sectionName := item.SectionName
			innerWidth := leftPaneWidth - 2
			maxSectionWidth := innerWidth - len(" ┌─  ─") // approximate, keep it safe
			if maxSectionWidth < 1 {
				maxSectionWidth = 1
			}
			if utf8.RuneCountInString(sectionName) > maxSectionWidth {
				// Truncate by runes to avoid splitting UTF-8 sequences.
				r := []rune(sectionName)
				sectionName = string(r[:maxSectionWidth]) + "..."
			}
			// Section headers are not selectable, but we can still highlight them differently
			line = s.status.Bold(true).Foreground(lipgloss.Color("62")).Render(fmt.Sprintf(" ┌─ %s ─", sectionName))
		} else {
			// Render service
			var service *systemd.ServiceState
			if idx, ok := m.serviceMap[item.ServiceName]; ok && idx < len(m.services) {
				service = m.services[idx]
			}
			
			statusIndicator := "?"
			enabled := false
			disabled := false
			otherUnitState := false
			if service != nil {
				statusIndicator = service.GetStateIndicator()
				switch service.UnitFileState {
				case "enabled":
					enabled = true
				case "disabled":
					disabled = true
				default:
					otherUnitState = true
				}
			}

			// Render enabled as a double-ring circle around the existing active/inactive circle.
			// When disabled, keep the original single-ring indicator (no outer circle).
			indicator := statusIndicator
			if enabled {
				if statusIndicator == "○" {
					indicator = "◎"
				} else if statusIndicator == "●" {
					// Use a slightly larger "bullseye" style glyph so the active enabled
					// marker doesn't appear smaller than the previous `●`.
					indicator = "◉"
				} else {
					// For non-circle status indicators, show an explicit enabled marker.
					indicator = statusIndicator + "E"
				}
			} else if disabled {
				// For non-circle status indicators, show an explicit disabled marker.
				if otherUnitState || (statusIndicator != "○" && statusIndicator != "●") {
					indicator = statusIndicator + "D"
				}
			} else {
				// Unknown/other unit file state: keep just the active-state indicator to avoid clutter.
				// (still '?' when we can't resolve)
			}

			name := item.ServiceName
			// Ensure the service name fits the bordered inner width.
			innerWidth := leftPaneWidth - 2
			indicatorWidth := utf8.RuneCountInString(indicator)
			// Format: " " + indicator + " " + name
			maxNameWidth := innerWidth - 2 - indicatorWidth
			if maxNameWidth < 1 {
				maxNameWidth = 1
			}
			if utf8.RuneCountInString(name) > maxNameWidth {
				r := []rune(name)
				name = string(r[:maxNameWidth]) + "..."
			}

			if i == m.selectedIndex {
				// If we're currently showing `systemctl cat` for this service, tint the selected row.
				if m.rightMode == rightPaneCat {
					line = s.selectedCat.Render(fmt.Sprintf(" %s %s", indicator, name))
				} else {
					line = s.selected.Render(fmt.Sprintf(" %s %s", indicator, name))
				}
			} else {
				line = s.normal.Render(fmt.Sprintf(" %s %s", indicator, name))
			}
		}

		lines = append(lines, line)
	}

	// Pad to fill height
	for len(lines) < innerHeight {
		lines = append(lines, "")
	}

	content := strings.Join(lines[:innerHeight], "\n")
	leftBorder := s.border
	if m.focus == focusLeft && !m.helpMode {
		leftBorder = s.focusedBorder
	}
	return leftBorder.
		Width(leftPaneWidth).
		Height(height).
		Render(content)
}

// renderLogs renders the right pane with logs
func (m *Model) renderLogs(height int) string {
	s := m.styles()
	innerHeight := height - 2
	if innerHeight < 1 {
		innerHeight = 1
	}

	// Optional follow-mode header line (only when following).
	followHeaderLines := 0
	if m.followMode {
		followHeaderLines = 1
	}

	logLinesCapacity := innerHeight - followHeaderLines
	if logLinesCapacity < 1 {
		logLinesCapacity = 1
	}

	start := 0
	if m.focus == focusRight && m.logTop >= 0 && !m.followMode {
		// When focused, treat logTop as "end index" for the window.
		top := m.logTop
		if top > len(m.logLines) {
			top = len(m.logLines)
		}
		start = top - logLinesCapacity
		if start < 0 {
			start = 0
		}
	} else {
		// Default to tail.
		start = len(m.logLines) - logLinesCapacity
		if start < 0 {
			start = 0
		}
	}

	var lines []string
	if m.followMode {
		lines = append(lines, s.status.Render(" [FOLLOW MODE]"))
	} else if len(m.logLines) == 0 {
		lines = append(lines, s.normal.Render("No logs available"))
	}

	for _, line := range m.logLines[start:] {
		// Truncate long lines
		maxWidth := m.width - leftPaneWidth - 6
		if len(line) > maxWidth {
			line = line[:maxWidth] + "..."
		}
		lines = append(lines, s.normal.Render(line))
	}

	// Pad to fill the inner border height.
	for len(lines) < innerHeight {
		lines = append(lines, "")
	}

	content := strings.Join(lines[:innerHeight], "\n")

	rightBorder := s.border
	if m.focus == focusRight && !m.helpMode {
		rightBorder = s.focusedBorder
	}
	return rightBorder.
		Width(m.width - leftPaneWidth - 2).
		Height(height).
		Render(content)
}

// renderCat renders the right pane with `systemctl cat` output for the selected unit.
func (m *Model) renderCat(height int) string {
	s := m.styles()
	innerHeight := height - 2
	if innerHeight < 1 {
		innerHeight = 1
	}

	if len(m.catLines) == 0 {
		rightBorder := s.border
		if m.focus == focusRight && !m.helpMode {
			rightBorder = s.focusedBorder
		}
		return rightBorder.
			Width(m.width - leftPaneWidth - 2).
			Height(height).
			Render("No unit file available")
	}

	start := m.catTop
	if start < 0 {
		start = 0
	}
	if start > len(m.catLines) {
		start = len(m.catLines)
	}

	var lines []string
	for _, line := range m.catLines[start:] {
		// Truncate long lines
		maxWidth := m.width - leftPaneWidth - 6
		if maxWidth < 1 {
			maxWidth = 1
		}
		if len(line) > maxWidth {
			line = line[:maxWidth] + "..."
		}
		lines = append(lines, s.normal.Render(line))
		if len(lines) >= innerHeight {
			break
		}
	}

	for len(lines) < innerHeight {
		lines = append(lines, "")
	}

	content := strings.Join(lines[:innerHeight], "\n")
	rightBorder := s.border
	if m.focus == focusRight && !m.helpMode {
		rightBorder = s.focusedBorder
	}
	return rightBorder.
		Width(m.width - leftPaneWidth - 2).
		Height(height).
		Render(content)
}

// renderFooter renders the footer with keybindings and status
func (m *Model) renderFooter() string {
	s := m.styles()
	keyMap := DefaultKeyMap
	keys := []string{
		keyMap.Help.Help().Key + ":" + keyMap.Help.Help().Desc,
		keyMap.Up.Help().Key + ":" + keyMap.Up.Help().Desc,
		keyMap.Down.Help().Key + ":" + keyMap.Down.Help().Desc,
		keyMap.Start.Help().Key + ":" + keyMap.Start.Help().Desc,
		keyMap.Stop.Help().Key + ":" + keyMap.Stop.Help().Desc,
		keyMap.Restart.Help().Key + ":" + keyMap.Restart.Help().Desc,
		keyMap.Reload.Help().Key + ":" + keyMap.Reload.Help().Desc,
		keyMap.Refresh.Help().Key + ":" + keyMap.Refresh.Help().Desc,
		keyMap.Enter.Help().Key + ":" + keyMap.Enter.Help().Desc,
		keyMap.Enable.Help().Key + ":" + keyMap.Enable.Help().Desc,
		keyMap.Disable.Help().Key + ":" + keyMap.Disable.Help().Desc,
		keyMap.DaemonReload.Help().Key + ":" + keyMap.DaemonReload.Help().Desc,
		keyMap.Theme.Help().Key + ":" + keyMap.Theme.Help().Desc,
		keyMap.Follow.Help().Key + ":" + keyMap.Follow.Help().Desc,
		keyMap.Quit.Help().Key + ":" + keyMap.Quit.Help().Desc,
	}

	keybindings := strings.Join(keys, " | ")

	// Status message
	status := m.statusMessage
	if strings.Contains(status, "Failed") || strings.Contains(status, "Error") {
		status = s.error.Render(status)
	} else if strings.Contains(status, "Successfully") {
		status = s.success.Render(status)
	} else {
		status = s.status.Render(status)
	}

	// Truncate if too long
	maxWidth := m.width - 2
	if len(keybindings) > maxWidth-20 {
		keybindings = keybindings[:maxWidth-20] + "..."
	}

	footerLine1 := keybindings
	footerLine2 := status

	return s.footer.
		Width(m.width).
		Render(footerLine1 + "\n" + footerLine2)
}
