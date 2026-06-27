package tui

import "charm.land/lipgloss/v2"

var styles = struct {
	brand    lipgloss.Style
	muted    lipgloss.Style
	selected lipgloss.Style
	error    lipgloss.Style
	success  lipgloss.Style
	panel    lipgloss.Style
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

	success: lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true),

	panel: lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("238")).
		Padding(0, 1),
}
