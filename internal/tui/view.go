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
	if m.width > 0 && m.height > 0 && (m.width < minWidth || m.height < minHeight) {
		return tea.NewView(fmt.Sprintf(
			"%s\n\nTerminal too small: %dx%d\nResize to at least %dx%d.\n\n[q] quit\n",
			m.renderHeader(),
			m.width,
			m.height,
			minWidth,
			minHeight,
		))
	}

	var s strings.Builder
	s.WriteString(m.renderHeader() + "\n\n")

	if m.connected {
		fmt.Fprintf(&s, "Agent connected \n")
		if m.status.State != "" {
			fmt.Fprintf(&s, "Status: %s\n", m.status.State)
		}
		if m.status.Error != "" {
			fmt.Fprintf(&s, "Error: %s\n", styles.error.Render(m.status.Error))
		}

		s.WriteString("\nStations\n")
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
	} else {
		s.WriteString(styles.muted.Render("Agent status: not connected.") + "\n\n")
		s.WriteString("Start the local agent, then reconnect with:\n\n")
		fmt.Fprintf(&s, "  ssh -p %s -R 127.0.0.1:%d:127.0.0.1:17777 127.0.0.1\n", sshPort, agentForwardPort)
	}

	if m.message != "" {
		fmt.Fprintf(&s, "\nDetail: %s\n", m.message)
	}

	s.WriteString("\n" + m.renderFooter() + "\n")
	return tea.NewView(s.String())
}
