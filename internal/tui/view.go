package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

const (
	minWidth  = 80
	minHeight = 24
)

func (m Model) View() tea.View {
	if m.width > 0 && m.height > 0 && (m.width < minWidth || m.height < minHeight) {
		return newView(fmt.Sprintf(
			"%s\n\nTerminal too small: %dx%d\nResize to at least %dx%d.\n\n[q] quit\n",
			m.renderHeader(),
			m.width,
			m.height,
			minWidth,
			minHeight,
		))
	}

	var s strings.Builder
	s.WriteString(m.renderHeader() + "\n\n")
	s.WriteString(m.renderContent())

	s.WriteString("\n" + m.renderFooter())
	return newView(s.String())
}

func newView(content string) tea.View {
	view := tea.NewView(content)
	view.AltScreen = true
	return view
}
