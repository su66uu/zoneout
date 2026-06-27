package tui

import (
	"fmt"
	"strings"
)

func (m Model) renderStations() string {
	var s strings.Builder

	s.WriteString("Stations\n")
	for i, station := range m.stations {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}

		row := fmt.Sprintf("%s %s", cursor, station.Name)
		if i == m.cursor {
			row = styles.selected.Render(row)
		}
		fmt.Fprintln(&s, row)
	}

	return s.String()
}

func (m Model) renderConsole() string {
	var s strings.Builder

	if m.connected {
		fmt.Fprintln(&s, "Agent connected")
		if m.status.State != "" {
			fmt.Fprintf(&s, "Status: %s\n", m.status.State)
		}
		if m.status.Error != "" {
			fmt.Fprintf(&s, "Error: %s\n", styles.error.Render(m.status.Error))
		}
		return s.String()
	}

	s.WriteString(styles.muted.Render("Agent status: not connected.") + "\n\n")
	s.WriteString("Start the local agent, then reconnect with:\n\n")
	fmt.Fprintf(&s, "  ssh -p %s -R 127.0.0.1:%d:127.0.0.1:17777 127.0.0.1\n", sshPort, agentForwardPort)

	return s.String()
}
