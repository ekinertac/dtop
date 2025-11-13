package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ekinertac/dtop/model"
)

const (
	// Column widths
	colNameWidth   = 50
	colStatusWidth = 30
	colCPUWidth    = 8
	colMemWidth    = 8
	colUptimeWidth = 12
)

var (
	// Colors
	primaryColor    = lipgloss.Color("#00D9FF")
	successColor    = lipgloss.Color("#00FF87")
	warningColor    = lipgloss.Color("#FFAF00")
	dangerColor     = lipgloss.Color("#FF5555")
	mutedColor      = lipgloss.Color("#6272A4")
	backgroundColor = lipgloss.Color("#282A36")
	foregroundColor = lipgloss.Color("#F8F8F2")

	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(mutedColor)

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#44475A")).
			Foreground(foregroundColor)

	projectStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor)

	containerStyle = lipgloss.NewStyle().
			Foreground(foregroundColor)

	runningStyle = lipgloss.NewStyle().
			Foreground(successColor)

	stoppedStyle = lipgloss.NewStyle().
			Foreground(dangerColor)

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2).
			Background(backgroundColor)

	menuItemStyle = lipgloss.NewStyle().
			Foreground(foregroundColor).
			PaddingLeft(2)

	menuSelectedStyle = lipgloss.NewStyle().
				Foreground(backgroundColor).
				Background(primaryColor).
				PaddingLeft(2)

	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1)
)

// truncateOrPad truncates or pads a string to a fixed width
func truncateOrPad(s string, width int) string {
	// Use rune count for proper Unicode handling
	runes := []rune(s)
	if len(runes) > width {
		return string(runes[:width-3]) + "..."
	}
	return s + strings.Repeat(" ", width-len(runes))
}

func (m Model) renderView() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	// Render based on view mode
	switch m.viewMode {
	case ViewModeLogs:
		return m.renderLogs()
	case ViewModeMenu:
		return m.renderMenu()
	}

	var content strings.Builder
	var footer strings.Builder

	// Title
	content.WriteString(titleStyle.Render("dtop - Docker Container Monitor"))
	content.WriteString("\n\n")

	// Header with fixed column widths
	header := truncateOrPad("NAME", colNameWidth) + " " +
		truncateOrPad("STATUS", colStatusWidth) + " " +
		truncateOrPad("CPU %", colCPUWidth) + " " +
		truncateOrPad("MEM %", colMemWidth) + " " +
		"UPTIME"
	content.WriteString(headerStyle.Render(header))
	content.WriteString("\n")

	// Calculate visible height (total - title/header - footer)
	// Title + blank = 2, Header = 1, Footer + blank = 2, Total overhead = 5
	visibleHeight := m.height - 5
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	// Tree view with viewport
	if m.tree != nil && len(m.tree.Flat) > 0 {
		// Calculate viewport boundaries
		viewportEnd := m.viewportTop + visibleHeight
		if viewportEnd > len(m.tree.Flat) {
			viewportEnd = len(m.tree.Flat)
		}

		// Render only visible items
		renderedLines := 0
		for i := m.viewportTop; i < viewportEnd; i++ {
			node := m.tree.Flat[i]
			line := m.renderNode(node, i == m.tree.Selected)
			content.WriteString(line)
			content.WriteString("\n")
			renderedLines++
		}

		// Fill remaining space with empty lines to push footer to bottom
		for renderedLines < visibleHeight {
			content.WriteString("\n")
			renderedLines++
		}

		// Add scroll indicator if there are more items
		totalItems := len(m.tree.Flat)
		if totalItems > visibleHeight {
			scrollInfo := fmt.Sprintf(" [%d-%d of %d]", m.viewportTop+1, viewportEnd, totalItems)
			footer.WriteString(lipgloss.NewStyle().Foreground(mutedColor).Render(scrollInfo))
			footer.WriteString(" ")
		}
	} else {
		content.WriteString("No containers found\n")
		// Fill space
		for i := 0; i < visibleHeight-1; i++ {
			content.WriteString("\n")
		}
	}

	// Help text (sticky footer)
	helpText := "↑↓/PgUp/PgDn:navigate  ←→:collapse/expand  enter:menu  q:quit"
	footer.WriteString(helpStyle.Render(helpText))

	return content.String() + "\n" + footer.String()
}

func (m Model) renderNode(node *model.TreeNode, selected bool) string {
	depth := m.tree.GetDepth(node)
	indent := strings.Repeat("  ", depth)

	var line string

	switch node.Type {
	case model.NodeTypeProject:
		icon := "▼"
		if !node.Expanded {
			icon = "▶"
		}
		projectName := fmt.Sprintf("%s %s (%d)", icon, node.Name, len(node.Children))
		fullText := indent + projectName
		
		// Pad to full row width for consistent selection highlight
		totalWidth := colNameWidth + 1 + colStatusWidth + 1 + colCPUWidth + 1 + colMemWidth + 1 + colUptimeWidth
		paddedText := truncateOrPad(fullText, totalWidth)
		
		if selected {
			line = selectedStyle.Render(paddedText)
		} else {
			line = projectStyle.Render(paddedText)
		}

	case model.NodeTypeContainer:
		if node.Container == nil {
			return ""
		}

		c := node.Container
		
		// Prepare each column with fixed width
		nameText := indent + "  " + c.Name
		name := truncateOrPad(nameText, colNameWidth)
		
		// Status column (apply color after padding)
		statusText := truncateOrPad(c.Status, colStatusWidth)
		var status string
		if c.State == "running" {
			status = runningStyle.Render(statusText)
		} else {
			status = stoppedStyle.Render(statusText)
		}
		
		cpu := truncateOrPad(fmt.Sprintf("%.1f%%", c.CPUPerc), colCPUWidth)
		mem := truncateOrPad(fmt.Sprintf("%.1f%%", c.MemPerc), colMemWidth)
		uptime := truncateOrPad(model.FormatUptime(c.CreatedAt), colUptimeWidth)

		// Build the full line
		if selected {
			// For selected rows, apply background to entire row using padded columns
			fullText := name + " " + statusText + " " + cpu + " " + mem + " " + uptime
			line = selectedStyle.Render(fullText)
		} else {
			// For unselected rows, apply colors per column
			line = containerStyle.Render(name) + " " + status + " " + 
				containerStyle.Render(cpu) + " " + 
				containerStyle.Render(mem) + " " + 
				containerStyle.Render(uptime)
		}
	}

	return line
}

func (m Model) renderMenu() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("dtop - Docker Container Monitor"))
	b.WriteString("\n\n")

	// Get selected node info for context
	node := m.tree.GetSelected()
	if node != nil {
		contextInfo := ""
		if node.Type == model.NodeTypeProject {
			contextInfo = fmt.Sprintf("Actions for project: %s", node.Name)
		} else if node.Container != nil {
			contextInfo = fmt.Sprintf("Actions for container: %s", node.Container.Name)
		}
		b.WriteString(projectStyle.Render(contextInfo))
		b.WriteString("\n\n")
	}

	// Menu items
	for i, item := range m.menuItems {
		prefix := "  "
		if i == m.menuSelected {
			prefix = "> "
			b.WriteString(menuSelectedStyle.Render(prefix + item.Label))
		} else {
			b.WriteString(menuItemStyle.Render(prefix + item.Label))
		}
		b.WriteString("\n")
	}

	// Help text
	b.WriteString("\n")
	helpText := "↑↓:select  enter:execute  esc:back"
	b.WriteString(helpStyle.Render(helpText))

	return b.String()
}

