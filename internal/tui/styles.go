package tui

import "charm.land/lipgloss/v2"

var styles = struct {
	brand    lipgloss.Style
	muted    lipgloss.Style
	selected lipgloss.Style
	error    lipgloss.Style
}{
	brand: lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true),

	muted: lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")),

	selected: lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true),

	error: lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true),
}
