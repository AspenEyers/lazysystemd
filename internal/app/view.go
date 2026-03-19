package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lazysystemd/internal/systemd"
)

var (
	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230"))

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250"))

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46"))

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Height(1)
)

// View renders the UI
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	// Calculate pane heights
	headerHeight := 1
	footerHeight := 2
	contentHeight := m.height - headerHeight - footerHeight

	if contentHeight < 1 {
		contentHeight = 1
	}

	// Left pane: services list
	leftPane := m.renderServicesList(contentHeight)

	// Right pane: logs
	rightPane := m.renderLogs(contentHeight)

	// Combine panes
	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	// Header
	header := titleStyle.Render("lazysystemd")

	// Footer
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		content,
		footer,
	)
}

// renderServicesList renders the left pane with the services list
func (m *Model) renderServicesList(height int) string {
	var lines []string

	for i, item := range m.items {
		var line string
		
		if item.IsSection {
			// Render section header
			sectionName := item.SectionName
			if len(sectionName) > leftPaneWidth-5 {
				sectionName = sectionName[:leftPaneWidth-5] + "..."
			}
			// Section headers are not selectable, but we can still highlight them differently
			line = statusStyle.Bold(true).Foreground(lipgloss.Color("62")).Render(fmt.Sprintf(" ┌─ %s ─", sectionName))
		} else {
			// Render service
			var service *systemd.ServiceState
			if idx, ok := m.serviceMap[item.ServiceName]; ok && idx < len(m.services) {
				service = m.services[idx]
			}
			
			indicator := "?"
			if service != nil {
				indicator = service.GetStateIndicator()
			}

			name := item.ServiceName
			if len(name) > leftPaneWidth-5 {
				name = name[:leftPaneWidth-5] + "..."
			}

			if i == m.selectedIndex {
				line = selectedStyle.Render(fmt.Sprintf(" %s %s", indicator, name))
			} else {
				line = normalStyle.Render(fmt.Sprintf(" %s %s", indicator, name))
			}
		}

		lines = append(lines, line)
	}

	// Pad to fill height
	for len(lines) < height {
		lines = append(lines, "")
	}

	content := strings.Join(lines[:height], "\n")
	return borderStyle.
		Width(leftPaneWidth).
		Height(height).
		Render(content)
}

// renderLogs renders the right pane with logs
func (m *Model) renderLogs(height int) string {
	if len(m.logLines) == 0 {
		return borderStyle.
			Width(m.width - leftPaneWidth - 2).
			Height(height).
			Render("No logs available")
	}

	// Show last N lines that fit
	visibleLines := height - 2
	if visibleLines < 1 {
		visibleLines = 1
	}

	start := len(m.logLines) - visibleLines
	if start < 0 {
		start = 0
	}

	var lines []string
	for _, line := range m.logLines[start:] {
		// Truncate long lines
		maxWidth := m.width - leftPaneWidth - 6
		if len(line) > maxWidth {
			line = line[:maxWidth] + "..."
		}
		lines = append(lines, normalStyle.Render(line))
	}

	// Pad to fill height
	for len(lines) < visibleLines {
		lines = append(lines, "")
	}

	content := strings.Join(lines[:visibleLines], "\n")
	
	// Add follow mode indicator
	header := ""
	if m.followMode {
		header = statusStyle.Render(" [FOLLOW MODE]")
	}

	return borderStyle.
		Width(m.width - leftPaneWidth - 2).
		Height(height).
		Render(header + "\n" + content)
}

// renderFooter renders the footer with keybindings and status
func (m *Model) renderFooter() string {
	keyMap := DefaultKeyMap
	keys := []string{
		keyMap.Up.Help().Key + ":" + keyMap.Up.Help().Desc,
		keyMap.Down.Help().Key + ":" + keyMap.Down.Help().Desc,
		keyMap.Start.Help().Key + ":" + keyMap.Start.Help().Desc,
		keyMap.Stop.Help().Key + ":" + keyMap.Stop.Help().Desc,
		keyMap.Restart.Help().Key + ":" + keyMap.Restart.Help().Desc,
		keyMap.Reload.Help().Key + ":" + keyMap.Reload.Help().Desc,
		keyMap.Follow.Help().Key + ":" + keyMap.Follow.Help().Desc,
		keyMap.Quit.Help().Key + ":" + keyMap.Quit.Help().Desc,
	}

	keybindings := strings.Join(keys, " | ")

	// Status message
	status := m.statusMessage
	if strings.Contains(status, "Failed") || strings.Contains(status, "Error") {
		status = errorStyle.Render(status)
	} else if strings.Contains(status, "Successfully") {
		status = successStyle.Render(status)
	} else {
		status = statusStyle.Render(status)
	}

	// Truncate if too long
	maxWidth := m.width - 2
	if len(keybindings) > maxWidth-20 {
		keybindings = keybindings[:maxWidth-20] + "..."
	}

	footerLine1 := keybindings
	footerLine2 := status

	return footerStyle.
		Width(m.width).
		Render(footerLine1 + "\n" + footerLine2)
}
