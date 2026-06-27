package main

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"zoneout/internal/agentclient"
	"zoneout/internal/tui"

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

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	client := agentclient.New(agentBaseURL)
	connected, message := checkAgentHealth(s.Context(), client)
	return tui.NewModel(client, connected, message), []tea.ProgramOption{}
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
