package ui

import (
	"fmt"
	"strings"
)

func (m Model) renderLogs() string {
	var b strings.Builder

	// Title
	title := fmt.Sprintf("dtop - Logs: %s", m.logsContainer)
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	// Split logs into lines
	lines := strings.Split(m.logsContent, "\n")
	
	// Calculate visible height
	visibleHeight := m.height - 4 // Title + blank + footer + blank

	// Clamp scroll position
	maxScroll := len(lines) - visibleHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.logsScroll > maxScroll {
		m.logsScroll = maxScroll
	}

	// Render visible lines
	end := m.logsScroll + visibleHeight
	if end > len(lines) {
		end = len(lines)
	}

	for i := m.logsScroll; i < end; i++ {
		b.WriteString(lines[i])
		b.WriteString("\n")
	}

	// Fill remaining space
	renderedLines := end - m.logsScroll
	for i := renderedLines; i < visibleHeight; i++ {
		b.WriteString("\n")
	}

	// Footer with scroll indicator
	footer := fmt.Sprintf("Lines %d-%d of %d", m.logsScroll+1, end, len(lines))
	b.WriteString(helpStyle.Render(footer))
	b.WriteString("  ")
	b.WriteString(helpStyle.Render("↑↓/PgUp/PgDn/g/G:scroll  q/esc:back"))

	return b.String()
}

