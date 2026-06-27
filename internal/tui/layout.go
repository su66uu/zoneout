package tui

import (
	"fmt"
	"strings"
	"time"
)

func (m Model) renderHeader() string {
	return styles.brand.Render("ZONEOUT") + " " + styles.muted.Render("Music over SSH")
}

func (m Model) renderFooter() string {
	left := "[p] play  [s] stop  [r] refresh  [q] quit"
	right := "Uptime: " + formatDuration(time.Since(m.startedAt))

	if m.width <= 0 || len(left)+len(right)+1 >= m.width {
		return left
	}

	return left + strings.Repeat(" ", m.width-len(left)-len(right)) + right
}

func formatDuration(d time.Duration) string {
	total := int(d.Seconds())
	hours := total / 3600
	minutes := (total % 3600) / 60
	seconds := total % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}
