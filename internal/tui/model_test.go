package tui

import (
	"strings"
	"testing"

	"zoneout/internal/agentclient"
	ztea "zoneout/internal/bubbletea"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestWindowSizeRendersSmallTerminalFallback(t *testing.T) {
	model := NewModel(nil, false, "")
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 79, Height: 20})
	model = updated.(Model)

	view := cleanView(model)
	if !strings.Contains(view, "Terminal too small: 79x20") {
		t.Fatalf("small terminal view did not include resize warning:\n%s", view)
	}
}

func TestMinimumViewportRendersBothPanesAndFooter(t *testing.T) {
	model := NewModel(nil, true, "")
	model.width = 80
	model.height = 24

	view := cleanView(model)
	for _, want := range []string{"STATIONS", "CONSOLE", "Awaiting playback command.", "zoneout  [p] play"} {
		if !strings.Contains(view, want) {
			t.Fatalf("minimum viewport did not contain %q:\n%s", want, view)
		}
	}
}

func TestViewsUseAlternateScreen(t *testing.T) {
	model := NewModel(nil, true, "")
	model.width = 120
	model.height = 30

	if view := model.View(); !view.AltScreen {
		t.Fatal("main view did not enable alternate screen")
	}

	model.width = 79
	model.height = 20
	if view := model.View(); !view.AltScreen {
		t.Fatal("small terminal view did not enable alternate screen")
	}
}

func TestFooterOmitsUptimeAndStylesShortcuts(t *testing.T) {
	model := NewModel(nil, true, "")

	footer := model.renderFooter()
	cleanFooter := ansi.Strip(footer)
	if strings.Contains(cleanFooter, "Uptime") {
		t.Fatalf("footer included uptime: %q", cleanFooter)
	}
	for _, want := range []string{"[p] play", "[s] stop", "[r] refresh", "[q] quit"} {
		if !strings.Contains(cleanFooter, want) {
			t.Fatalf("footer did not contain %q: %q", want, cleanFooter)
		}
	}
	if !strings.Contains(footer, styles.shortcut.Render("[p]")) {
		t.Fatalf("footer did not style shortcut token: %q", footer)
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

func TestDisconnectedViewIncludesReconnectInstructions(t *testing.T) {
	model := NewModel(nil, false, "expected ok but got connection refused")
	model.width = 120
	model.height = 40

	view := cleanView(model)
	for _, want := range []string{"Agent status: not connected.", "ssh -p 23234", "Detail: expected ok"} {
		if !strings.Contains(view, want) {
			t.Fatalf("disconnected view did not contain %q:\n%s", want, view)
		}
	}
}

func TestConnectingViewShowsSyntheticConsoleDetails(t *testing.T) {
	model := NewModel(nil, true, "")
	model.width = 120
	model.height = 40
	model.status = agentclient.StatusResponse{
		State:     "connecting",
		StreamURL: model.stations[0].URL,
	}

	view := cleanView(model)
	for _, want := range []string{"State: connecting", "Signal: connecting to Code Radio", "EQ:", "Activity:"} {
		if !strings.Contains(view, want) {
			t.Fatalf("connecting view did not contain %q:\n%s", want, view)
		}
	}
}

func TestWideViewIncludesStationAndConsolePanes(t *testing.T) {
	model := NewModel(nil, true, "")
	model.width = 120
	model.height = 30
	model.status = agentclient.StatusResponse{
		State:     "playing",
		StreamURL: model.stations[0].URL,
	}

	view := cleanView(model)
	for _, want := range []string{"STATIONS", "CONSOLE", "Agent: connected", "Signal: transmitting Code Radio", "Track: Code Radio", "EQ:", "on-air"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view did not contain %q:\n%s", want, view)
		}
	}
}

func TestErrorViewKeepsBackendErrorVisible(t *testing.T) {
	model := NewModel(nil, true, "")
	model.width = 120
	model.height = 40
	model.status = agentclient.StatusResponse{
		State: "error",
		Error: "stream decoder failed",
	}

	view := cleanView(model)
	for _, want := range []string{"State: error", "Error: stream decoder failed"} {
		if !strings.Contains(view, want) {
			t.Fatalf("error view did not contain %q:\n%s", want, view)
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

func TestPanelLinesTruncateLongText(t *testing.T) {
	lines := panelLines("TITLE", strings.Repeat("x", 80), 10, 2)
	if len(lines) != 2 {
		t.Fatalf("panelLines returned %d lines, want 2", len(lines))
	}

	for _, line := range lines {
		if width := ansi.StringWidth(ansi.Strip(line)); width > 10 {
			t.Fatalf("line width = %d, want <= 10: %q", width, line)
		}
	}
}

func TestFullHeightViewKeepsHeaderInFirstLine(t *testing.T) {
	model := NewModel(nil, true, "")
	model.width = 120
	model.height = 30
	model.status = agentclient.StatusResponse{
		State:     "playing",
		StreamURL: model.stations[0].URL,
	}

	view := cleanView(model)
	lines := strings.Split(view, "\n")
	if got := len(lines); got > model.height {
		t.Fatalf("view rendered %d lines, want <= %d:\n%s", got, model.height, view)
	}
	if !strings.Contains(lines[0], "ZONEOUT") {
		t.Fatalf("first line = %q, want app header", lines[0])
	}
}

func TestPlayingStatusDoesNotScheduleBackgroundTick(t *testing.T) {
	model := NewModel(nil, true, "")

	_, cmd := model.Update(ztea.AgentStatusMsg{
		Status: agentclient.StatusResponse{State: "playing"},
	})
	if cmd != nil {
		t.Fatal("playing status scheduled a command, want nil to avoid periodic redraw")
	}
}

func cleanView(model Model) string {
	return ansi.Strip(model.View().Content)
}
