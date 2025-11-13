package model

import (
	"sort"
	"strings"
	"time"

	"github.com/ekinertac/dtop/docker"
)

type NodeType int

const (
	NodeTypeProject NodeType = iota
	NodeTypeContainer
)

type TreeNode struct {
	Type      NodeType
	Name      string
	Container *docker.ContainerInfo
	Children  []*TreeNode
	Expanded  bool
	Parent    *TreeNode
}

type Tree struct {
	Root     *TreeNode
	Flat     []*TreeNode // Flattened view for navigation
	Selected int
}

// ParseProjectName extracts the project name from a container name
// Examples:
//   - myproject_web_1 -> myproject
//   - myproject-web-1 -> myproject
//   - standalone -> standalone (no prefix)
func ParseProjectName(containerName string) string {
	// Try underscore separator first (docker-compose v1)
	if idx := strings.Index(containerName, "_"); idx != -1 {
		return containerName[:idx]
	}
	
	// Try dash separator (docker-compose v2)
	if idx := strings.Index(containerName, "-"); idx != -1 {
		return containerName[:idx]
	}
	
	// No separator found, use full name as project
	return containerName
}

// BuildTree groups containers by project prefix
func BuildTree(containers []docker.ContainerInfo) *Tree {
	root := &TreeNode{
		Type:     NodeTypeProject,
		Name:     "root",
		Expanded: true,
		Children: []*TreeNode{},
	}

	// Group containers by project
	projects := make(map[string][]*docker.ContainerInfo)
	for i := range containers {
		projectName := ParseProjectName(containers[i].Name)
		projects[projectName] = append(projects[projectName], &containers[i])
	}

	// Sort project names alphabetically
	projectNames := make([]string, 0, len(projects))
	for name := range projects {
		projectNames = append(projectNames, name)
	}
	sort.Strings(projectNames)

	// Build tree structure in alphabetical order
	for _, projectName := range projectNames {
		containers := projects[projectName]
		
		// Sort containers within project alphabetically
		sort.Slice(containers, func(i, j int) bool {
			return containers[i].Name < containers[j].Name
		})

		projectNode := &TreeNode{
			Type:     NodeTypeProject,
			Name:     projectName,
			Expanded: true,
			Parent:   root,
			Children: []*TreeNode{},
		}

		for _, container := range containers {
			containerNode := &TreeNode{
				Type:      NodeTypeContainer,
				Name:      container.Name,
				Container: container,
				Parent:    projectNode,
			}
			projectNode.Children = append(projectNode.Children, containerNode)
		}

		root.Children = append(root.Children, projectNode)
	}

	tree := &Tree{
		Root:     root,
		Selected: 0,
	}
	tree.UpdateFlatView()

	return tree
}

// UpdateFlatView creates a flattened view of visible nodes for navigation
func (t *Tree) UpdateFlatView() {
	t.Flat = []*TreeNode{}
	t.flattenNode(t.Root, 0)
}

func (t *Tree) flattenNode(node *TreeNode, depth int) {
	// Don't add root to flat view
	if node.Type != NodeTypeProject || node.Name != "root" {
		t.Flat = append(t.Flat, node)
	}

	// Add children if expanded
	if node.Expanded {
		for _, child := range node.Children {
			t.flattenNode(child, depth+1)
		}
	}
}

// GetSelected returns the currently selected node
func (t *Tree) GetSelected() *TreeNode {
	if t.Selected < 0 || t.Selected >= len(t.Flat) {
		return nil
	}
	return t.Flat[t.Selected]
}

// MoveUp moves selection up
func (t *Tree) MoveUp() {
	if t.Selected > 0 {
		t.Selected--
	}
}

// MoveDown moves selection down
func (t *Tree) MoveDown() {
	if t.Selected < len(t.Flat)-1 {
		t.Selected++
	}
}

// ToggleExpanded toggles the expanded state of the selected node
func (t *Tree) ToggleExpanded() {
	node := t.GetSelected()
	if node != nil && node.Type == NodeTypeProject {
		node.Expanded = !node.Expanded
		t.UpdateFlatView()
	}
}

// GetDepth returns the depth of a node in the tree
func (t *Tree) GetDepth(node *TreeNode) int {
	depth := 0
	current := node.Parent
	for current != nil && current.Name != "root" {
		depth++
		current = current.Parent
	}
	return depth
}

// GetNodePath returns a unique path identifier for a node
func (t *Tree) GetNodePath(node *TreeNode) string {
	if node == nil {
		return ""
	}
	
	// Build path from root to node
	path := []string{}
	current := node
	for current != nil && current.Name != "root" {
		path = append([]string{current.Name}, path...)
		current = current.Parent
	}
	
	return strings.Join(path, "/")
}

// RestoreSelection attempts to restore selection to a node with the given path
func (t *Tree) RestoreSelection(path string) {
	if path == "" {
		return
	}
	
	// Search through flat view for matching path
	for i, node := range t.Flat {
		if t.GetNodePath(node) == path {
			t.Selected = i
			return
		}
	}
	
	// If exact match not found, keep current selection (or default to 0)
	if t.Selected >= len(t.Flat) {
		t.Selected = 0
	}
}

// FormatUptime formats the container uptime
func FormatUptime(created time.Time) string {
	duration := time.Since(created)
	
	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60
	
	if days > 0 {
		return formatDuration(days, hours, minutes, "d", "h", "m")
	}
	if hours > 0 {
		return formatDuration(hours, minutes, 0, "h", "m", "")
	}
	return formatDuration(minutes, 0, 0, "m", "", "")
}

func formatDuration(a, b, c int, aUnit, bUnit, cUnit string) string {
	result := ""
	if a > 0 {
		result += formatUnit(a, aUnit)
	}
	if b > 0 {
		if result != "" {
			result += " "
		}
		result += formatUnit(b, bUnit)
	}
	if c > 0 && cUnit != "" {
		if result != "" {
			result += " "
		}
		result += formatUnit(c, cUnit)
	}
	return result
}

func formatUnit(value int, unit string) string {
	if unit == "" {
		return ""
	}
	return formatInt(value) + unit
}

func formatInt(value int) string {
	return string(rune('0' + value/10)) + string(rune('0' + value%10))
}

