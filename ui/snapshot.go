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
	header := fmt.Sprintf("%-50s %-30s %-8s %-8s %s",
		"NAME", "STATUS", "CPU %", "MEM %", "UPTIME")
	fmt.Println(header)
	fmt.Println(strings.Repeat("-", 120))

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
		name := truncateOrPadPlain(indent+"  "+c.Name, 50)
		status := truncateOrPadPlain(c.Status, 30)
		cpu := fmt.Sprintf("%-8s", fmt.Sprintf("%.1f%%", c.CPUPerc))
		mem := fmt.Sprintf("%-8s", fmt.Sprintf("%.1f%%", c.MemPerc))
		uptime := model.FormatUptime(c.CreatedAt)

		fmt.Printf("%s %s %s %s %s\n", name, status, cpu, mem, uptime)
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

