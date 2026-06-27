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
		case "p":
			if m.connected && m.client != nil && len(m.stations) > 0 {
				return m, ztea.PlayCmd(m.client, m.stations[m.cursor].URL)
			}
		case "s":
			if m.connected && m.client != nil && len(m.stations) > 0 {
				return m, ztea.StopCmd(m.client)
			}
		case "r":
			if m.connected && m.client != nil {
				return m, ztea.StatusCmd(m.client)
			}
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
			return m, nil
		}
		m.status = msg.Status
		m.message = ""

		if msg.Status.State == "connecting" && m.client != nil {
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
