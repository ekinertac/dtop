package ui

import (
	"fmt"
	"strings"

	"github.com/ekinertac/dtop/model"
)

// PrintSnapshot prints a non-interactive snapshot of the container tree
func PrintSnapshot(tree *model.Tree) {
	// Title
	fmt.Println("dtop - Docker Container Monitor")
	fmt.Println()

	// Header
	header := fmt.Sprintf("%-40s %-25s %-12s %-12s %-14s %s",
		"NAME", "STATUS", "CPU", "MEMORY", "NET RX/TX", "UPTIME")
	fmt.Println(header)
	fmt.Println(strings.Repeat("-", 130))

	if tree == nil || len(tree.Flat) == 0 {
		fmt.Println("No containers found")
		return
	}

	// Print all nodes
	for _, node := range tree.Flat {
		printNode(tree, node)
	}
}

func printNode(tree *model.Tree, node *model.TreeNode) {
	depth := tree.GetDepth(node)
	indent := strings.Repeat("  ", depth)

	switch node.Type {
	case model.NodeTypeProject:
		icon := "▼"
		if !node.Expanded {
			icon = "▶"
		}
		projectName := fmt.Sprintf("%s %s (%d)", icon, node.Name, len(node.Children))
		fmt.Println(indent + projectName)

	case model.NodeTypeContainer:
		if node.Container == nil {
			return
		}

		c := node.Container
		name := truncateOrPadPlain(indent+"  "+c.Name, 40)
		status := truncateOrPadPlain(c.Status, 25)
		
		// CPU with bar
		cpuBar := renderProgressBarPlain(c.CPUPerc, 5)
		cpuText := fmt.Sprintf("%3.0f%% %s", c.CPUPerc, cpuBar)
		cpu := truncateOrPadPlain(cpuText, 12)
		
		// Memory with bar
		memBar := renderProgressBarPlain(c.MemPerc, 5)
		memText := fmt.Sprintf("%3.0f%% %s", c.MemPerc, memBar)
		mem := truncateOrPadPlain(memText, 12)
		
		// Network
		netRx := formatNetBytesPlain(c.NetRx)
		netTx := formatNetBytesPlain(c.NetTx)
		netText := fmt.Sprintf("%s/%s", netRx, netTx)
		net := truncateOrPadPlain(netText, 14)
		
		uptime := model.FormatUptime(c.CreatedAt)

		fmt.Printf("%s %s %s %s %s %s\n", name, status, cpu, mem, net, uptime)
	}
}

// truncateOrPadPlain truncates or pads a string to a fixed width (plain text, no ANSI)
func truncateOrPadPlain(s string, width int) string {
	runes := []rune(s)
	if len(runes) > width {
		return string(runes[:width-3]) + "..."
	}
	return s + strings.Repeat(" ", width-len(runes))
}

// renderProgressBarPlain creates a simple progress bar (plain text)
func renderProgressBarPlain(percent float64, width int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filled := int((percent / 100.0) * float64(width))
	if filled > width {
		filled = width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return bar
}

// formatNetBytesPlain formats network bytes with units
func formatNetBytesPlain(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return "0"
	}
	
	div := uint64(unit)
	exp := 0
	for n := bytes / unit; n >= unit && exp < 4; n /= unit {
		div *= unit
		exp++
	}
	
	value := float64(bytes) / float64(div)
	units := []string{"B", "K", "M", "G", "T"}
	
	if value >= 100 {
		return fmt.Sprintf("%.0f%s", value, units[exp])
	} else if value >= 10 {
		return fmt.Sprintf("%.1f%s", value, units[exp])
	}
	return fmt.Sprintf("%.1f%s", value, units[exp])
}

