package tui

import (
	"fmt"
	"strings"
	"time"

	"zoneout/internal/agentclient"

	ztea "zoneout/internal/bubbletea"

	tea "charm.land/bubbletea/v2"
)

type Station struct {
	Name string
	URL  string
}

type Model struct {
	connected bool
	message   string
	stations  []Station
	client    *agentclient.Client
	status    agentclient.StatusResponse
	cursor    int
	startedAt time.Time
	width     int
	height    int
	tick      int
}

const (
	sshPort          = "23234"
	agentForwardPort = uint32(27777)
)

func NewModel(client *agentclient.Client, connected bool, message string) Model {
	return Model{
		connected: connected,
		message:   message,
		client:    client,
		startedAt: time.Now(),
		stations: []Station{
			{
				Name: "Code Radio",
				URL:  "https://coderadio-admin-v2.freecodecamp.org/listen/coderadio/radio.mp3",
			},
		},
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

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

func (m Model) View() tea.View {
	var s strings.Builder
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
