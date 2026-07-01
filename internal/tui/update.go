package tui

import (
	"time"

	ztea "zoneout/internal/bubbletea"

	tea "charm.land/bubbletea/v2"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.stations)-1 {
				m.cursor++
			}
		case "enter", "p":
			return m.playSelected()
		case "s":
			if m.connected && m.client != nil {
				m.notice = "stop requested"
				return m, ztea.StopCmd(m.client)
			}
			m.notice = "agent unavailable"
		case "r":
			if m.connected && m.client != nil {
				m.notice = "refresh requested"
				return m, ztea.StatusCmd(m.client)
			}
			m.notice = "agent unavailable"
		case "ctrl+c", "q":
			if m.connected && m.client != nil {
				return m, tea.Sequence(
					ztea.StopCmd(m.client),
					tea.Quit,
				)
			}
			return m, tea.Quit
		}
	case ztea.AgentStatusMsg:
		if msg.Err != nil {
			m.message = msg.Err.Error()
			m.notice = "agent error"
			return m, nil
		}
		m.status = msg.Status
		m.message = ""
		m.notice = statusNotice(msg.Status.State)

		if m.client != nil && (msg.Status.State == "connecting" || msg.Status.State == "playing") {
			return m, ztea.DelayedStatusCmd(300 * time.Millisecond)
		}
		return m, nil
	case ztea.DelayedStatusMsg:
		if m.connected && m.client != nil {
			return m, ztea.StatusCmd(m.client)
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}
	return m, nil
}

func (m Model) playSelected() (tea.Model, tea.Cmd) {
	if !m.connected || m.client == nil {
		m.notice = "agent unavailable"
		return m, nil
	}

	if len(m.stations) == 0 {
		m.notice = "no stations available"
		return m, nil
	}

	station := m.stations[m.cursor]
	if !station.Enabled || station.URL == "" {
		m.notice = "station unavailable in this build"
		return m, nil
	}

	m.notice = "play requested"
	return m, ztea.PlayCmd(m.client, station.URL)
}

func statusNotice(state string) string {
	switch state {
	case "connecting":
		return "connecting"
	case "playing":
		return "transmitting"
	case "idle":
		return "awaiting command"
	case "error":
		return "playback error"
	default:
		return ""
	}
}
