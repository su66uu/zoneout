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

	btcommands "zoneout/btCommands"
	"zoneout/internal/agentclient"

	tea "charm.land/bubbletea/v2"
	"charm.land/log/v2"
	"charm.land/wish/v2"
	"charm.land/wish/v2/activeterm"
	"charm.land/wish/v2/bubbletea"
	"charm.land/wish/v2/logging"
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
	Name string
	URL  string
}

type model struct {
	connected bool
	message   string
	stations  []station
	client    *agentclient.Client
	status    agentclient.StatusResponse
	cursor    int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "p":
			if m.connected && m.client != nil && len(m.stations) > 0 {
				return m, btcommands.PlayCmd(m.client, m.stations[m.cursor].URL)
			}
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	case btcommands.AgentStatusMsg:
		if msg.Err != nil {
			m.message = msg.Err.Error()
			return m, nil
		}
		m.status = msg.Status
		m.message = ""
	}
	return m, nil
}

func (m model) View() tea.View {
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

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	client := agentclient.New(agentBaseURL)
	connected, message := checkAgentHealth(s.Context(), client)
	return model{
		connected: connected,
		message:   message,
		client:    client,
		stations: []station{
			{
				Name: "Code Radio",
				URL:  "https://coderadio-admin-v2.freecodecamp.org/listen/coderadio/radio.mp3",
			},
		},
	}, []tea.ProgramOption{}
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
