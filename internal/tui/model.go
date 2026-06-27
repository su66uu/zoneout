package tui

import (
	"time"

	"zoneout/internal/agentclient"
	ztea "zoneout/internal/bubbletea"

	tea "charm.land/bubbletea/v2"
)

type Station struct {
	Name    string
	URL     string
	Title   string
	Artist  string
	Enabled bool
}

type Model struct {
	connected bool
	message   string
	notice    string
	stations  []Station
	client    *agentclient.Client
	status    agentclient.StatusResponse
	cursor    int
	startedAt time.Time
	width     int
	height    int
	tick      int
}

type tickMsg time.Time

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
		stations:  defaultStations(),
	}
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{tickCmd()}
	if m.connected && m.client != nil {
		cmds = append(cmds, ztea.StatusCmd(m.client))
	}
	return tea.Batch(cmds...)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func defaultStations() []Station {
	return []Station{
		{
			Name:    "Code Radio",
			URL:     "https://coderadio-admin-v2.freecodecamp.org/listen/coderadio/radio.mp3",
			Title:   "Code Radio",
			Artist:  "freeCodeCamp",
			Enabled: true,
		},
		{
			Name:   "Synthwave Grid",
			Title:  "Neon Compile",
			Artist: "Zoneout Preview",
		},
		{
			Name:   "Night Drive 24/7",
			Title:  "Late Build Lane",
			Artist: "Zoneout Preview",
		},
		{
			Name:   "Minimalist Focus",
			Title:  "Quiet Loop",
			Artist: "Zoneout Preview",
		},
		{
			Name:   "Deep Techno Lab",
			Title:  "Runtime Pressure",
			Artist: "Zoneout Preview",
		},
		{
			Name:   "Ambient Void",
			Title:  "Blank Terminal",
			Artist: "Zoneout Preview",
		},
	}
}
