package tui

import (
	"strings"
	"testing"
	"time"

	"zoneout/internal/agentclient"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestFormatDuration(t *testing.T) {
	got := formatDuration(2*time.Hour + 3*time.Minute + 4*time.Second)
	if got != "02:03:04" {
		t.Fatalf("formatDuration() = %q, want %q", got, "02:03:04")
	}
}

func TestWindowSizeRendersSmallTerminalFallback(t *testing.T) {
	model := NewModel(nil, false, "")
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 79, Height: 20})
	model = updated.(Model)

	view := cleanView(model)
	if !strings.Contains(view, "Terminal too small: 79x20") {
		t.Fatalf("small terminal view did not include resize warning:\n%s", view)
	}
}

func TestCursorNavigationStaysWithinStations(t *testing.T) {
	model := NewModel(nil, false, "")

	updated, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	model = updated.(Model)
	if model.cursor != 0 {
		t.Fatalf("cursor after moving above first station = %d, want 0", model.cursor)
	}

	for range len(model.stations) + 3 {
		updated, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		model = updated.(Model)
	}

	last := len(model.stations) - 1
	if model.cursor != last {
		t.Fatalf("cursor after moving below last station = %d, want %d", model.cursor, last)
	}
}

func TestDisabledStationDoesNotIssuePlayCommand(t *testing.T) {
	model := NewModel(agentclient.New("http://127.0.0.1:1"), true, "")
	model.cursor = 1

	updated, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model = updated.(Model)

	if cmd != nil {
		t.Fatal("disabled station returned a command, want nil")
	}
	if model.notice != "station unavailable in this build" {
		t.Fatalf("notice = %q, want disabled station notice", model.notice)
	}
}

func TestWideViewIncludesStationAndConsolePanes(t *testing.T) {
	model := NewModel(nil, true, "")
	model.width = 120
	model.height = 30
	model.tick = 3
	model.status = agentclient.StatusResponse{
		State:     "playing",
		StreamURL: model.stations[0].URL,
	}

	view := cleanView(model)
	for _, want := range []string{"STATIONS", "CONSOLE", "Agent: connected", "Track: Code Radio", "EQ:", "on-air"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view did not contain %q:\n%s", want, view)
		}
	}
}

func TestStackedViewKeepsBothPanes(t *testing.T) {
	model := NewModel(nil, true, "")
	model.width = 90
	model.height = 30

	view := cleanView(model)
	for _, want := range []string{"STATIONS", "CONSOLE", "Awaiting playback command."} {
		if !strings.Contains(view, want) {
			t.Fatalf("stacked view did not contain %q:\n%s", want, view)
		}
	}
}

func cleanView(model Model) string {
	return ansi.Strip(model.View().Content)
}
