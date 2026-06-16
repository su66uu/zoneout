package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	connected, message := checkAgentHealth(s.Context())
	return model{
		connected: connected,
		message:   message,
	}, []tea.ProgramOption{}
}

type model struct {
	connected bool
	message   string
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() tea.View {
	var s strings.Builder
	s.WriteString("Zoneout Phase 1: Reverse Tunnel Check\n\n")

	if m.connected {
		s.WriteString("Agent status: connected through SSH reverse tunnel.\n")
	} else {
		s.WriteString("Agent status: not connected.\n\n")
		s.WriteString("Start the local agent, then reconnect with:\n\n")
		fmt.Fprintf(&s, "  ssh -p %s -R 127.0.0.1:%d:127.0.0.1:17777 127.0.0.1\n", sshPort, agentForwardPort)
	}

	if m.message != "" {
		fmt.Fprintf(&s, "\nDetail: %s\n", m.message)
	}

	s.WriteString("\nPress q to quit.\n")
	return tea.NewView(s.String())
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

func checkAgentHealth(ctx context.Context) (bool, string) {
	reqCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, agentHealthURL, nil)
	if err != nil {
		return false, err.Error()
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err.Error()
	}
	defer func() { _ = res.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(res.Body, 128))
	if err != nil {
		return false, err.Error()
	}

	if res.StatusCode != http.StatusOK {
		return false, fmt.Sprintf("health check returned %s", res.Status)
	}

	text := strings.TrimSpace(string(body))
	if text != "ok" {
		return false, fmt.Sprintf("unexpected health response %q", text)
	}

	return true, "received ok from " + agentHealthURL
}
