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
		if !station.Enabled {
			row = styles.muted.Render(row)
		}
		if m.isActiveStation(station) {
			row += " " + styles.success.Render("on-air")
		}
		if i == m.cursor {
			row = styles.selected.Render(row)
		}
		fmt.Fprintln(&s, row)
	}

	fmt.Fprintf(&s, "\n%s\n", styles.muted.Render(m.stationSummary()))
	return s.String()
}

func (m Model) stationSummary() string {
	online := 0
	for _, station := range m.stations {
		if station.Enabled {
			online++
		}
	}

	previews := len(m.stations) - online
	return fmt.Sprintf("%d channel online - %d previews", online, previews)
}

func (m Model) isActiveStation(station Station) bool {
	return station.Enabled && station.URL != "" && station.URL == m.status.StreamURL && m.status.State == "playing"
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
