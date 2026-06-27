package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	headerHeight = 1
	headerGap    = 1
	footerHeight = 1
	panelGap     = 2
	panelChrome  = 4
)

func (m Model) renderHeader() string {
	return styles.brand.Render("ZONEOUT") + " " + styles.muted.Render("Music over SSH")
}

func (m Model) renderFooter() string {
	left := "zoneout  [p] play  [s] stop  [r] refresh  [q] quit"
	right := "Uptime: " + formatDuration(time.Since(m.startedAt))

	if m.width <= 0 || len(left)+len(right)+1 >= m.width {
		return left
	}

	return left + strings.Repeat(" ", m.width-len(left)-len(right)) + right
}

func (m Model) renderContent() string {
	if m.width <= 0 || m.height <= 0 {
		return m.renderConsole() + "\n" + m.renderStations()
	}

	contentHeight := m.contentHeight()
	if contentHeight <= 0 {
		return ""
	}

	if m.shouldStackPanes() {
		return m.renderStackedContent(contentHeight)
	}

	return m.renderWideContent(contentHeight)
}

func (m Model) contentHeight() int {
	return m.height - headerHeight - headerGap - footerHeight
}

func (m Model) shouldStackPanes() bool {
	return m.width < 100
}

func (m Model) renderWideContent(height int) string {
	leftWidth := 32
	if m.width < 120 {
		leftWidth = 30
	}

	rightWidth := m.width - leftWidth - panelGap
	if rightWidth < 20 {
		return m.renderStackedContent(height)
	}

	left := renderPanel("STATIONS", m.renderStations(), leftWidth, height)
	right := renderPanel("CONSOLE", m.renderConsole(), rightWidth, height)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", panelGap), right)
}

func (m Model) renderStackedContent(height int) string {
	if height < 6 {
		return ansi.Truncate(m.renderConsole(), m.width, "")
	}

	stationsHeight := height / 2
	consoleHeight := height - stationsHeight

	stations := renderPanel("STATIONS", m.renderStations(), m.width, stationsHeight)
	console := renderPanel("CONSOLE", m.renderConsole(), m.width, consoleHeight)
	return lipgloss.JoinVertical(lipgloss.Left, stations, console)
}

func renderPanel(title string, body string, width int, height int) string {
	if width < panelChrome {
		width = panelChrome
	}
	if height < 2 {
		height = 2
	}

	innerWidth := width - panelChrome
	innerHeight := height - 2

	lines := panelLines(title, body, innerWidth, innerHeight)
	content := strings.Join(lines, "\n")
	return styles.panel.Width(width).Render(content)
}

func panelLines(title string, body string, width int, height int) []string {
	lines := make([]string, 0, height)
	if height <= 0 {
		return lines
	}

	lines = append(lines, styles.muted.Render(ansi.Truncate(title, width, "")))

	bodyLines := strings.Split(strings.TrimRight(body, "\n"), "\n")
	for _, line := range bodyLines {
		if len(lines) >= height {
			break
		}
		lines = append(lines, ansi.Truncate(line, width, ""))
	}

	for len(lines) < height {
		lines = append(lines, "")
	}
	return lines
}

func formatDuration(d time.Duration) string {
	total := int(d.Seconds())
	hours := total / 3600
	minutes := (total % 3600) / 60
	seconds := total % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}
