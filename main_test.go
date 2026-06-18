package main

import (
	"errors"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"zoneout/internal/agentclient"
	ztea "zoneout/internal/bubbletea"
)

func TestRenderFitsCommonTerminalSizes(t *testing.T) {
	tests := []struct {
		name      string
		width     int
		height    int
		connected bool
		help      bool
		status    string
		want      []string
	}{
		{
			name:      "wide connected dashboard",
			width:     120,
			height:    32,
			connected: true,
			want:      []string{"zoneout", "now playing", "Code Radio", "stations"},
		},
		{
			name:      "wide playing dashboard",
			width:     96,
			height:    24,
			connected: true,
			status:    playbackStatePlaying,
			want:      []string{"stream: playing", "signal:"},
		},
		{
			name:      "medium stacked dashboard",
			width:     72,
			height:    20,
			connected: true,
			want:      []string{"now playing", "stations"},
		},
		{
			name:      "narrow connected dashboard",
			width:     48,
			height:    14,
			connected: true,
			want:      []string{"Code Radio", "space/enter"},
		},
		{
			name:      "narrow setup",
			width:     40,
			height:    12,
			connected: false,
			want:      []string{"first run setup", "ssh -p"},
		},
		{
			name:      "connected help",
			width:     80,
			height:    24,
			connected: true,
			help:      true,
			want:      []string{"Zoneout controls", "Toggle this help"},
		},
		{
			name:      "setup help",
			width:     60,
			height:    18,
			connected: false,
			help:      true,
			want:      []string{"Manual setup", "reverse tunnel"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newModel(nil, tt.connected, "")
			m.width = tt.width
			m.height = tt.height
			m.help.ShowAll = tt.help
			m.status.State = tt.status
			m.resizeList()

			rendered := m.render()
			assertRenderFits(t, rendered, tt.width, tt.height)
			plain := ansi.Strip(rendered)
			for _, want := range tt.want {
				if !strings.Contains(plain, want) {
					t.Fatalf("rendered UI missing %q\n%s", want, rendered)
				}
			}
		})
	}
}

func assertRenderFits(t *testing.T, rendered string, width, height int) {
	t.Helper()

	lines := strings.Split(strings.TrimRight(rendered, "\n"), "\n")
	if len(lines) > height {
		t.Fatalf("rendered height %d exceeds terminal height %d\n%s", len(lines), height, rendered)
	}
	for i, line := range lines {
		if lineWidth := lipgloss.Width(line); lineWidth > width {
			t.Fatalf("line %d width %d exceeds terminal width %d: %q", i+1, lineWidth, width, line)
		}
	}
}

func TestStatusErrorMarksAgentDisconnected(t *testing.T) {
	m := newModel(nil, true, "")
	m.status = agentclient.StatusResponse{
		State:     playbackStatePlaying,
		StreamURL: "https://example.com/radio.mp3",
	}

	updated, cmd := m.Update(ztea.AgentStatusMsg{Err: errors.New("connection refused")})
	if cmd != nil {
		t.Fatalf("unexpected command after status failure")
	}

	next := updated.(model)
	if next.connected {
		t.Fatalf("expected agent to be marked disconnected")
	}
	if next.status.State != "" || next.status.StreamURL != "" {
		t.Fatalf("expected stale playback status to be cleared: %#v", next.status)
	}
	if !strings.Contains(next.message, "agent status failed: connection refused") {
		t.Fatalf("expected status failure message, got %q", next.message)
	}
}

func TestAgentHealthFailureMessage(t *testing.T) {
	message := agentHealthErrorMessage(errors.New("connection refused"))
	if !strings.Contains(message, "expected ok but got ") {
		t.Fatalf("expected readable health failure message, got %q", message)
	}
	if strings.Contains(message, "gotunexpected") {
		t.Fatalf("expected health failure message to include spacing, got %q", message)
	}
}
