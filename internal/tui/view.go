package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

const (
	minWidth  = 80
	minHeight = 24
)

func (m Model) View() tea.View {
	var s strings.Builder

	if m.width > 0 && m.height > 0 && (m.width < minWidth || m.height < minHeight) {
		return tea.NewView(fmt.Sprintf(
			"Zoneout\n\nTerminal too small: %dx%d\nResize to at least %dx%d.\n\n[q] quit\n",
			m.width,
			m.height,
			minWidth,
			minHeight,
		))
	}

	s.WriteString("Zoneout\n\n")

	if m.connected {
		fmt.Fprintf(&s, "Agent connected \n")
		if m.status.State != "" {
			fmt.Fprintf(&s, "Status: %s\n", m.status.State)
		}
		if m.status.Error != "" {
			fmt.Fprintf(&s, "Error: %s\n", m.status.Error)
		}

		s.WriteString("\nStations\n")
		for i, station := range m.stations {
			cursor := " "
			if i == m.cursor {
				cursor = ">"
			}
			fmt.Fprintf(&s, "%s %s\n", cursor, station.Name)
		}
	} else {
		s.WriteString("Agent status: not connected.\n\n")
		s.WriteString("Start the local agent, then reconnect with:\n\n")
		fmt.Fprintf(&s, "  ssh -p %s -R 127.0.0.1:%d:127.0.0.1:17777 127.0.0.1\n", sshPort, agentForwardPort)
	}

	if m.message != "" {
		fmt.Fprintf(&s, "\nDetail: %s\n", m.message)
	}

	s.WriteString("\n[p] play  [s] stop  [r] refresh  [q] quit\n")
	return tea.NewView(s.String())
}
