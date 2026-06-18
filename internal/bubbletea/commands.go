package bubbletea

import (
	"context"
	"time"

	"zoneout/internal/agentclient"

	tea "charm.land/bubbletea/v2"
)

type AgentStatusMsg struct {
	Status agentclient.StatusResponse
	Err    error
}

func StatusCmd(c *agentclient.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		status, err := c.Status(ctx)
		return AgentStatusMsg{Status: status, Err: err}
	}
}

func PlayCmd(c *agentclient.Client, streamURL string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		status, err := c.Play(ctx, streamURL)
		return AgentStatusMsg{Status: status, Err: err}
	}
}

func StopCmd(c *agentclient.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		status, err := c.Stop(ctx)
		return AgentStatusMsg{Status: status, Err: err}
	}
}
