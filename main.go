package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"zoneout/internal/agentclient"
	ztea "zoneout/internal/bubbletea"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/log/v2"
	"charm.land/wish/v2"
	"charm.land/wish/v2/activeterm"
	"charm.land/wish/v2/bubbletea"
	"charm.land/wish/v2/logging"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/ssh"
)

const (
	sshHost          = "127.0.0.1"
	sshPort          = "23234"
	agentForwardHost = "127.0.0.1"
	agentForwardPort = uint32(27777)
	agentBaseURL     = "http://127.0.0.1:27777"
	agentHealthURL   = "http://127.0.0.1:27777/health"
)

const (
	playbackStateConnecting = "connecting"
	playbackStatePlaying    = "playing"
)

const (
	connectingStatusPollDelay = 300 * time.Millisecond
	playingStatusPollDelay    = 2 * time.Second
)

func main() {
	forwardHandler := &ssh.ForwardedTCPHandler{}

	srv, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(sshHost, sshPort)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		withReverseForwarding(forwardHandler),
		wish.WithMiddleware(
			bubbletea.Middleware(teaHandler),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Error("Could not start server", "error", err)
		os.Exit(1)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		log.Info("Starting the SSH server", "host", sshHost, "port", sshPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start the server", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("Stopping the server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("Could not stop the server", "error", err)
	}
}

type station struct {
	Name    string
	URL     string
	Tagline string
	Genre   string
	Codec   string
	Live    bool
}

func (s station) Title() string {
	return s.Name
}

func (s station) Description() string {
	var parts []string
	if s.Tagline != "" {
		parts = append(parts, s.Tagline)
	}
	if s.Codec != "" {
		parts = append(parts, s.Codec)
	}
	if s.Genre != "" {
		parts = append(parts, s.Genre)
	}
	if s.Live {
		parts = append(parts, "live")
	}
	return strings.Join(parts, "  ·  ")
}

func (s station) FilterValue() string {
	return strings.Join([]string{s.Name, s.Tagline, s.Genre, s.Codec}, " ")
}

type keyMap struct {
	Play    key.Binding
	Stop    key.Binding
	Refresh key.Binding
	Filter  key.Binding
	Help    key.Binding
	Quit    key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Play: key.NewBinding(
			key.WithKeys(" ", "enter", "p"),
			key.WithHelp("space/enter", "play"),
		),
		Stop: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "stop"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Play, k.Stop, k.Refresh, k.Filter, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Play, k.Stop, k.Refresh},
		{k.Filter, k.Help, k.Quit},
	}
}

type frameMsg struct{}

type agentHealthMsg struct {
	Connected bool
	Message   string
}

type model struct {
	connected   bool
	message     string
	client      *agentclient.Client
	status      agentclient.StatusResponse
	width       int
	height      int
	frame       int
	keys        keyMap
	help        help.Model
	stationList list.Model
	spinner     spinner.Model
}

func (m model) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, func() tea.Msg { return tea.RequestWindowSize() })
	if m.connected && m.client != nil {
		cmds = append(cmds, ztea.StatusCmd(m.client))
	}
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeList()
		return m, nil
	case tea.KeyPressMsg:
		if key.Matches(msg, m.keys.Quit) {
			if m.connected && m.client != nil {
				return m, tea.Sequence(
					ztea.StopCmd(m.client),
					tea.Quit,
				)
			}
			return m, tea.Quit
		}

		if m.stationList.SettingFilter() {
			return m.updateStationList(msg)
		}

		switch {
		case key.Matches(msg, m.keys.Play):
			if station, ok := m.selectedStation(); ok && m.connected && m.client != nil {
				return m, tea.Batch(
					ztea.PlayCmd(m.client, station.URL),
					m.spinner.Tick,
					animationTick(),
				)
			}
		case key.Matches(msg, m.keys.Stop):
			if m.connected && m.client != nil {
				return m, ztea.StopCmd(m.client)
			}
		case key.Matches(msg, m.keys.Refresh):
			if m.client != nil {
				if m.connected {
					return m, ztea.StatusCmd(m.client)
				}
				return m, healthCmd(m.client)
			}
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		}
		return m.updateStationList(msg)
	case ztea.AgentStatusMsg:
		if msg.Err != nil {
			m.message = msg.Err.Error()
			return m, nil
		}
		m.status = msg.Status
		m.message = ""

		if delay, ok := statusPollDelay(msg.Status.State); ok && m.client != nil {
			return m, tea.Batch(
				ztea.DelayedStatusCmd(delay),
				m.spinner.Tick,
				animationTick(),
			)
		}
		return m, nil
	case ztea.DelayedStatusMsg:
		if m.connected && m.client != nil {
			return m, ztea.StatusCmd(m.client)
		}
		return m, nil
	case agentHealthMsg:
		m.connected = msg.Connected
		m.message = msg.Message
		if m.connected && m.client != nil {
			return m, ztea.StatusCmd(m.client)
		}
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if m.isActive() {
			return m, cmd
		}
		return m, nil
	case frameMsg:
		m.frame++
		if m.isActive() {
			return m, animationTick()
		}
		return m, nil
	}
	return m, nil
}

func (m model) View() tea.View {
	content := m.render()
	view := tea.NewView(content)
	view.AltScreen = true
	view.WindowTitle = "zoneout"
	return view
}

func (m model) updateStationList(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.stationList, cmd = m.stationList.Update(msg)
	return m, cmd
}

func (m *model) resizeList() {
	width, height := m.listSize()
	m.stationList.SetSize(width, height)
	m.help.SetWidth(max(20, width))
}

func (m model) listSize() (int, int) {
	width := defaultWidth(m.width)
	height := defaultHeight(m.height)

	if width >= 96 && height >= 24 {
		return max(30, width-6), max(6, height-18)
	}
	if width >= 72 {
		return max(30, width-6), max(6, height-16)
	}
	return max(24, width-4), max(5, height-10)
}

func (m model) selectedStation() (station, bool) {
	item := m.stationList.SelectedItem()
	if item == nil {
		return station{}, false
	}
	station, ok := item.(station)
	return station, ok
}

func (m model) isActive() bool {
	return m.status.State == playbackStateConnecting || m.status.State == playbackStatePlaying
}

func (m model) render() string {
	width := defaultWidth(m.width)
	height := defaultHeight(m.height)

	header := m.renderHeader(width)
	footer := m.renderFooter(width)
	bodyHeight := max(6, height-lipgloss.Height(header)-lipgloss.Height(footer)-2)
	body := m.renderBody(width, bodyHeight)
	body = clipBlockHeight(body, bodyHeight)

	content := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(content)
}

func (m model) renderHeader(width int) string {
	status := "agent: missing"
	if m.connected {
		status = "agent: connected"
	}
	stream := "stream: idle"
	if m.status.State != "" {
		stream = "stream: " + m.status.State
	}
	title := accentStyle.Render("zoneout")
	line := fmt.Sprintf("%s  ssh session: zoneout.local  %s  %s  output: local audio", title, status, stream)

	return headerStyle.Width(width).Render(truncate(line, width-2))
}

func (m model) renderFooter(width int) string {
	footerHelpModel := m.help
	if footerHelpModel.ShowAll {
		footerHelpModel.ShowAll = false
	}
	footerHelp := footerHelpModel.View(m.keys)
	if m.stationList.SettingFilter() {
		footerHelp = dimStyle.Render("esc clear filter") + "  " + footerHelp
	}
	return footerStyle.Width(width).Render(truncate(footerHelp, width-2))
}

func (m model) renderBody(width, height int) string {
	if width < 54 || height <= 10 {
		return m.renderCompact(width, height)
	}
	if m.help.ShowAll {
		return m.renderHelpPanel(width, height)
	}
	if !m.connected {
		return m.renderSetup(width, height)
	}

	if width >= 96 && height >= 16 {
		return m.renderWide(width, height)
	}
	return m.renderStacked(width, height)
}

func (m model) renderWide(width, height int) string {
	innerWidth := width - 4
	topHeight := min(10, max(7, height/2))
	leftWidth := max(28, innerWidth/3)
	rightWidth := max(36, innerWidth-leftWidth-1)

	art := panelStyle.
		Width(leftWidth).
		Height(topHeight).
		Render(m.renderDeskArt(leftWidth, topHeight))
	nowPlaying := panelStyle.
		Width(rightWidth).
		Height(topHeight).
		Render(m.renderNowPlaying(rightWidth, topHeight))
	top := lipgloss.JoinHorizontal(lipgloss.Top, art, nowPlaying)

	listHeight := max(6, height-lipgloss.Height(top)-1)
	stations := m.renderStations(innerWidth+1, listHeight)
	return lipgloss.JoinVertical(lipgloss.Left, top, stations)
}

func (m model) renderCompact(width, height int) string {
	var s strings.Builder
	if !m.connected {
		s.WriteString(accentStyle.Render("first run setup"))
		s.WriteString("\n")
		fmt.Fprintf(&s, "ssh -p %s -R 127.0.0.1:%d:127.0.0.1:17777 127.0.0.1\n", sshPort, agentForwardPort)
		s.WriteString("r refresh  ? help  q quit")
		return panelStyle.
			Width(max(30, width-2)).
			Height(max(3, height-4)).
			Render(s.String())
	}

	state := "idle"
	if m.status.State != "" {
		state = m.status.State
	}
	station, ok := m.selectedStation()
	if !ok {
		s.WriteString("no station selected")
	} else {
		s.WriteString(dimStyle.Render("now playing"))
		s.WriteString("\n")
		s.WriteString(titleStyle.Render(station.Name))
	}
	s.WriteString("\n")
	fmt.Fprintf(&s, "state: %s", state)
	if state == playbackStatePlaying {
		fmt.Fprintf(&s, "\nsignal: %s", equalizer(m.frame))
	}
	return panelStyle.
		Width(max(30, width-2)).
		Height(max(3, height-4)).
		Render(s.String())
}

func (m model) renderStacked(width, height int) string {
	innerWidth := width - 2
	nowHeight := 8
	if width < 72 {
		nowHeight = 5
	}
	nowPlaying := panelStyle.
		Width(innerWidth).
		Height(nowHeight).
		Render(m.renderNowPlaying(innerWidth, nowHeight))
	listHeight := max(5, height-lipgloss.Height(nowPlaying)-1)
	stations := m.renderStations(innerWidth, listHeight)
	return lipgloss.JoinVertical(lipgloss.Left, nowPlaying, stations)
}

func (m model) renderStations(width, height int) string {
	stationList := m.stationList
	stationList.SetSize(max(24, width-2), max(4, height-2))
	return panelStyle.Width(width).Height(height).Render(stationList.View())
}

func clipBlockHeight(block string, height int) string {
	if height <= 0 {
		return ""
	}
	lines := strings.Split(strings.TrimRight(block, "\n"), "\n")
	if len(lines) <= height {
		return block
	}
	return strings.Join(lines[:height], "\n")
}

func (m model) renderSetup(width, height int) string {
	var s strings.Builder
	s.WriteString(accentStyle.Render("first run setup"))
	s.WriteString("\n\nZoneout needs a local audio agent because an SSH session cannot play sound on your machine by itself.\n\n")
	s.WriteString("Start the local agent and reconnect with this reverse tunnel:\n\n")
	fmt.Fprintf(&s, "  ssh -p %s -R 127.0.0.1:%d:127.0.0.1:17777 127.0.0.1\n", sshPort, agentForwardPort)
	if m.message != "" {
		fmt.Fprintf(&s, "\n%s %s\n", warningStyle.Render("detail:"), m.message)
	}
	s.WriteString("\nPress r after connecting the agent, ? for manual setup, or q to leave.")

	return panelStyle.
		Width(max(30, width-2)).
		Height(height).
		Render(s.String())
}

func (m model) renderHelpPanel(width, height int) string {
	markdown := connectedHelpMarkdown()
	if !m.connected {
		markdown = setupHelpMarkdown()
	}
	content := renderMarkdown(markdown, max(24, width-8))
	return panelStyle.
		Width(max(30, width-2)).
		Height(height).
		Render(lipgloss.NewStyle().MaxHeight(max(1, height-2)).Render(content))
}

func (m model) renderNowPlaying(width, height int) string {
	station, ok := m.selectedStation()
	if !ok {
		return "no station selected"
	}

	var s strings.Builder
	s.WriteString(dimStyle.Render("now playing"))
	s.WriteString("\n")
	s.WriteString(titleStyle.Render(station.Name))
	s.WriteString("\n")
	s.WriteString(dimStyle.Render(station.Description()))
	s.WriteString("\n\n")

	state := "idle"
	if m.status.State != "" {
		state = m.status.State
	}
	if state == playbackStateConnecting {
		fmt.Fprintf(&s, "%s %s\n", m.spinner.View(), "connecting to stream")
	} else {
		fmt.Fprintf(&s, "state: %s\n", state)
	}
	if state == playbackStatePlaying {
		fmt.Fprintf(&s, "signal: %s\n", equalizer(m.frame))
	}
	fmt.Fprintf(&s, "tunnel: 127.0.0.1:%d -> agent\n", agentForwardPort)
	if m.status.Error != "" {
		fmt.Fprintf(&s, "%s %s\n", warningStyle.Render("error:"), m.status.Error)
	}
	if m.message != "" {
		fmt.Fprintf(&s, "%s %s\n", warningStyle.Render("detail:"), m.message)
	}

	return lipgloss.NewStyle().MaxHeight(height).MaxWidth(width).Render(s.String())
}

func (m model) renderDeskArt(width, height int) string {
	art := []string{
		"      .-----------------.",
		"      |  ~/work         |",
		"      |  git diff       |",
		"      |  make test      |",
		"      '-----------------'",
		"          /|  headphones",
		"         /_|",
	}
	if width < 34 || height < 8 {
		art = []string{
			"~/work",
			"git diff",
			"make test",
		}
	}
	return dimStyle.Render(strings.Join(art, "\n"))
}

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	client := agentclient.New(agentBaseURL)
	connected, message := checkAgentHealth(s.Context(), client)
	return newModel(client, connected, message), []tea.ProgramOption{}
}

func defaultStations() []station {
	return []station{
		{
			Name:    "Code Radio",
			URL:     "https://coderadio-admin-v2.freecodecamp.org/listen/coderadio/radio.mp3",
			Tagline: "24/7 music designed for coding",
			Genre:   "focus",
			Codec:   "mp3",
			Live:    true,
		},
	}
}

func newStationList(stations []station) list.Model {
	items := make([]list.Item, len(stations))
	for i, station := range stations {
		items[i] = station
	}
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("#7DD3FC"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("#FBBF24"))
	stationList := list.New(items, delegate, 0, 0)
	stationList.Title = "stations"
	stationList.DisableQuitKeybindings()
	stationList.SetShowHelp(false)
	stationList.SetStatusBarItemName("station", "stations")
	return stationList
}

func newModel(client *agentclient.Client, connected bool, message string) model {
	spin := spinner.New(spinner.WithSpinner(spinner.Meter), spinner.WithStyle(accentStyle))
	helpModel := help.New()
	helpModel.Styles.ShortKey = helpModel.Styles.ShortKey.Foreground(lipgloss.Color("#7DD3FC"))
	return model{
		connected:   connected,
		message:     message,
		client:      client,
		keys:        newKeyMap(),
		help:        helpModel,
		stationList: newStationList(defaultStations()),
		spinner:     spin,
	}
}

func withReverseForwarding(forwardHandler *ssh.ForwardedTCPHandler) ssh.Option {
	return func(s *ssh.Server) error {
		s.ReversePortForwardingCallback = func(ctx ssh.Context, host string, port uint32) bool {
			allowed := host == agentForwardHost && port == agentForwardPort
			if allowed {
				log.Info("Accepted reverse forward", "host", host, "port", port)
			} else {
				log.Warn("Rejected reverse forward", "host", host, "port", port)
			}
			return allowed
		}
		s.RequestHandlers = map[string]ssh.RequestHandler{
			"tcpip-forward":        forwardHandler.HandleSSHRequest,
			"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
		}
		return nil
	}
}

func checkAgentHealth(ctx context.Context, client *agentclient.Client) (bool, string) {
	err := client.Health(ctx)
	if err != nil {
		return false, "expected ok but got" + err.Error()
	}

	return true, "received ok from " + agentHealthURL
}

func animationTick() tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(time.Time) tea.Msg {
		return frameMsg{}
	})
}

func healthCmd(client *agentclient.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		connected, message := checkAgentHealth(ctx, client)
		return agentHealthMsg{Connected: connected, Message: message}
	}
}

func renderMarkdown(markdown string, width int) string {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(max(24, width)),
	)
	if err != nil {
		return markdown
	}
	out, err := renderer.Render(markdown)
	if err != nil {
		return markdown
	}
	return strings.TrimSpace(out)
}

func connectedHelpMarkdown() string {
	return fmt.Sprintf(`# Zoneout controls

Zoneout is an SSH music player for coding sessions. The TUI runs over SSH; audio plays through your local agent.

## Keys

| Key | Action |
| --- | --- |
| space / enter / p | Play selected station |
| s | Stop playback |
| r | Refresh agent status |
| / | Filter stations |
| ? | Toggle this help |
| q / ctrl+c | Stop and quit |

## Status

- Agent: local control process reached through the reverse SSH tunnel.
- Stream: current playback state reported by the agent.
- Tunnel: server port %d forwards to your local agent.

If playback fails, refresh first. If the agent is missing, reconnect with the reverse tunnel shown in setup.
`, agentForwardPort)
}

func setupHelpMarkdown() string {
	return fmt.Sprintf(`# Manual setup

Zoneout needs a local audio agent because SSH cannot play sound on your machine by itself.

Run the agent locally, then connect with a reverse tunnel:

~~~sshconfig
ssh -p %s -R 127.0.0.1:%d:127.0.0.1:17777 127.0.0.1
~~~

Expected checks:

- The agent listens on "127.0.0.1:17777".
- The SSH server accepts "127.0.0.1:%d" as a reverse forward.
- Press "r" in Zoneout after the tunnel is available.

For a permanent setup, add a managed SSH config block that starts the agent and opens the reverse tunnel before launching the TUI.
`, sshPort, agentForwardPort, agentForwardPort)
}

func statusPollDelay(state string) (time.Duration, bool) {
	switch state {
	case playbackStateConnecting:
		return connectingStatusPollDelay, true
	case playbackStatePlaying:
		return playingStatusPollDelay, true
	default:
		return 0, false
	}
}

func equalizer(frame int) string {
	frames := []string{
		"▂▄▆█▆▄▂  ▃▅▇▅▃  ▂▄▆▄▂",
		"▃▅▇▅▃  ▂▄▆█▆▄▂  ▃▅▇▅▃",
		"▄▆█▆▄▂  ▃▅▇▅▃  ▂▄▆█▆▄",
		"▆█▆▄▂  ▂▄▆▄▂  ▃▅▇▅▃",
	}
	return frames[frame%len(frames)]
}

func defaultWidth(width int) int {
	if width <= 0 {
		return 96
	}
	return width
}

func defaultHeight(height int) int {
	if height <= 0 {
		return 28
	}
	return height
}

func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	if width <= 3 {
		return string(runes[:width])
	}
	return string(runes[:width-3]) + "..."
}

var (
	accentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7DD3FC")).
			Bold(true)
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB")).
			Bold(true)
	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF"))
	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FBBF24"))
	headerStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("#374151")).
			Padding(0, 1)
	footerStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(lipgloss.Color("#374151")).
			Padding(0, 1)
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#374151")).
			Padding(1, 2)
)
