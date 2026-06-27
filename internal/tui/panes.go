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
	return station.Enabled && station.URL != "" && station.URL == m.status.StreamURL && m.isStreaming()
}

func (m Model) renderConsole() string {
	var s strings.Builder

	if m.connected {
		state := m.playbackState()
		station := m.activeStation()

		fmt.Fprintln(&s, "Agent: connected")
		fmt.Fprintf(&s, "State: %s\n", m.renderState(state))

		if station.Name != "" {
			fmt.Fprintf(&s, "Station: %s\n", station.Name)
		}
		if m.isStreaming() {
			if state == "connecting" {
				fmt.Fprintf(&s, "Signal: connecting to %s\n", station.Name)
			} else {
				fmt.Fprintf(&s, "Signal: transmitting %s\n", station.Name)
			}
			if station.Title != "" {
				fmt.Fprintf(&s, "Track: %s\n", station.Title)
			}
			if station.Artist != "" {
				fmt.Fprintf(&s, "Artist: %s\n", station.Artist)
			}
			fmt.Fprintf(&s, "\n%s\n", m.equalizer())
			fmt.Fprintf(&s, "Activity: %s\n", m.progressBar(24))
		} else if state == "idle" || state == "ready" {
			fmt.Fprintf(&s, "\n%s\n", styles.muted.Render("Awaiting playback command."))
		}
		if m.status.Error != "" {
			fmt.Fprintf(&s, "Error: %s\n", styles.error.Render(m.status.Error))
		}
		m.writeConsoleMessages(&s)
		return s.String()
	}

	s.WriteString(styles.muted.Render("Agent status: not connected.") + "\n\n")
	s.WriteString("Start the local agent, then reconnect with:\n\n")
	fmt.Fprintf(&s, "  ssh -p %s -R 127.0.0.1:%d:127.0.0.1:17777 127.0.0.1\n", sshPort, agentForwardPort)
	m.writeConsoleMessages(&s)

	return s.String()
}

func (m Model) playbackState() string {
	if m.status.State != "" {
		return m.status.State
	}
	if m.connected {
		return "ready"
	}
	return "offline"
}

func (m Model) renderState(state string) string {
	switch state {
	case "playing":
		return styles.success.Render("playing")
	case "connecting":
		return styles.selected.Render("connecting")
	case "error":
		return styles.error.Render("error")
	case "idle", "ready":
		return styles.muted.Render(state)
	default:
		return state
	}
}

func (m Model) isStreaming() bool {
	return m.status.State == "playing" || m.status.State == "connecting"
}

func (m Model) activeStation() Station {
	if m.status.StreamURL != "" {
		for _, station := range m.stations {
			if station.URL == m.status.StreamURL {
				return station
			}
		}
	}
	return m.selectedStation()
}

func (m Model) selectedStation() Station {
	if len(m.stations) == 0 || m.cursor < 0 || m.cursor >= len(m.stations) {
		return Station{}
	}
	return m.stations[m.cursor]
}

func (m Model) equalizer() string {
	levels := []int{1, 3, 2, 4, 2, 5, 3, 4}
	var bars strings.Builder
	for i, level := range levels {
		shifted := ((level + m.tick + i) % 5) + 1
		if i > 0 {
			bars.WriteString(" ")
		}
		bars.WriteString(strings.Repeat("|", shifted))
	}
	return "EQ: " + styles.success.Render(bars.String())
}

func (m Model) progressBar(width int) string {
	if width < 4 {
		width = 4
	}

	filled := 1
	if m.status.State == "playing" {
		filled = (m.tick % width) + 1
	} else if m.status.State == "connecting" {
		filled = ((m.tick % width) / 2) + 1
	}

	if filled > width {
		filled = width
	}

	return "[" + strings.Repeat("#", filled) + strings.Repeat("-", width-filled) + "]"
}

func (m Model) writeConsoleMessages(s *strings.Builder) {
	if m.message != "" {
		fmt.Fprintf(s, "\nDetail: %s\n", m.message)
	}
	if m.notice != "" {
		fmt.Fprintf(s, "\n> %s\n", m.notice)
	}
}
