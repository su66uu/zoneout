package tui

import (
	"time"

	"zoneout/internal/agentclient"

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
