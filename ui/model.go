package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ekinertac/dtop/docker"
	"github.com/ekinertac/dtop/model"
)

type ViewMode int

const (
	ViewModeMain ViewMode = iota
	ViewModeMenu
	ViewModeLogs
)

type Model struct {
	dockerClient   *docker.Client
	tree           *model.Tree
	viewMode       ViewMode
	menuItems      []MenuItem
	menuSelected   int
	logsContent    string
	logsScroll     int
	logsContainer  string
	width          int
	height         int
	viewportTop    int // First visible line in the tree
	err            error
}

type MenuItem struct {
	Label  string
	Action func() tea.Cmd
}

type tickMsg time.Time

func NewModel(dockerClient *docker.Client) Model {
	return Model{
		dockerClient:  dockerClient,
		tree:          &model.Tree{},
		viewMode:      ViewModeMain,
		menuSelected:  0,
		logsScroll:    0,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.refreshContainersWithStats(false), // First load without stats (instant)
		tickCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) refreshContainers() tea.Cmd {
	return m.refreshContainersWithStats(true)
}

func (m Model) refreshContainersWithStats(includeStats bool) tea.Cmd {
	return func() tea.Msg {
		containers, err := m.dockerClient.ListContainersWithStats(includeStats)
		if err != nil {
			return errMsg{err}
		}
		return containersMsg(containers)
	}
}

type containersMsg []docker.ContainerInfo
type logsMsg struct {
	containerName string
	content       string
}
type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.adjustViewport() // Adjust viewport on resize
		return m, nil

	case containersMsg:
		// Preserve selection and expand/collapse state across refresh
		var selectedPath string
		expandedProjects := make(map[string]bool)
		
		if m.tree != nil {
			selectedNode := m.tree.GetSelected()
			if selectedNode != nil {
				selectedPath = m.tree.GetNodePath(selectedNode)
			}
			
			// Save expand/collapse state for each project
			for _, node := range m.tree.Flat {
				if node.Type == model.NodeTypeProject {
					expandedProjects[node.Name] = node.Expanded
				}
			}
		}
		
		m.tree = model.BuildTree(msg)
		
		// Restore expand/collapse state
		for _, node := range m.tree.Root.Children {
			if node.Type == model.NodeTypeProject {
				if expanded, exists := expandedProjects[node.Name]; exists {
					node.Expanded = expanded
				}
			}
		}
		m.tree.UpdateFlatView()
		
		// Restore selection if possible
		if selectedPath != "" {
			m.tree.RestoreSelection(selectedPath)
		}
		
		// Adjust viewport to ensure selection is visible
		m.adjustViewport()
		
		return m, nil

	case tickMsg:
		return m, tea.Batch(
			m.refreshContainers(),
			tickCmd(),
		)

	case logsMsg:
		m.logsContainer = msg.containerName
		m.logsContent = msg.content
		m.logsScroll = 0
		m.viewMode = ViewModeLogs
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	}

	return m, nil
}

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle logs view
	if m.viewMode == ViewModeLogs {
		switch msg.String() {
		case "esc", "q":
			m.viewMode = ViewModeMain
			m.logsContent = ""
			m.logsScroll = 0
		case "up", "k":
			if m.logsScroll > 0 {
				m.logsScroll--
			}
		case "down", "j":
			m.logsScroll++
		case "pgup":
			m.logsScroll -= m.height - 5
			if m.logsScroll < 0 {
				m.logsScroll = 0
			}
		case "pgdown":
			m.logsScroll += m.height - 5
		case "home":
			m.logsScroll = 0
		case "g":
			m.logsScroll = 0
		case "G":
			// Go to end
			m.logsScroll = 999999 // Will be clamped in view
		}
		return m, nil
	}

	// Handle menu navigation
	if m.viewMode == ViewModeMenu {
		switch msg.String() {
		case "up", "k":
			if m.menuSelected > 0 {
				m.menuSelected--
			}
		case "down", "j":
			if m.menuSelected < len(m.menuItems)-1 {
				m.menuSelected++
			}
		case "enter":
			// Execute selected action
			if m.menuSelected < len(m.menuItems) {
				cmd := m.menuItems[m.menuSelected].Action()
				m.viewMode = ViewModeMain
				return m, cmd
			}
		case "esc":
			m.viewMode = ViewModeMain
		}
		return m, nil
	}

	// Handle tree navigation
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "up", "k":
		m.tree.MoveUp()
		m.adjustViewport()

	case "down", "j":
		m.tree.MoveDown()
		m.adjustViewport()

	case "pgup":
		// Page up - move up by viewport height
		visibleHeight := m.height - 5
		if visibleHeight < 1 {
			visibleHeight = 1
		}
		for i := 0; i < visibleHeight && m.tree.Selected > 0; i++ {
			m.tree.MoveUp()
		}
		m.adjustViewport()

	case "pgdown":
		// Page down - move down by viewport height
		visibleHeight := m.height - 5
		if visibleHeight < 1 {
			visibleHeight = 1
		}
		for i := 0; i < visibleHeight && m.tree.Selected < len(m.tree.Flat)-1; i++ {
			m.tree.MoveDown()
		}
		m.adjustViewport()

	case "home":
		// Jump to top
		m.tree.Selected = 0
		m.adjustViewport()

	case "end":
		// Jump to bottom
		if len(m.tree.Flat) > 0 {
			m.tree.Selected = len(m.tree.Flat) - 1
		}
		m.adjustViewport()

	case "left", "h":
		node := m.tree.GetSelected()
		if node != nil && node.Type == model.NodeTypeProject && node.Expanded {
			node.Expanded = false
			m.tree.UpdateFlatView()
			m.adjustViewport()
		}

	case "right", "l":
		node := m.tree.GetSelected()
		if node != nil && node.Type == model.NodeTypeProject && !node.Expanded {
			node.Expanded = true
			m.tree.UpdateFlatView()
			m.adjustViewport()
		}

	case "enter":
		m.openMenu()
	}

	return m, nil
}

func (m *Model) openMenu() {
	node := m.tree.GetSelected()
	if node == nil {
		return
	}

	m.menuSelected = 0
	m.viewMode = ViewModeMenu

	switch node.Type {
	case model.NodeTypeProject:
		m.menuItems = m.getProjectMenuItems(node)
	case model.NodeTypeContainer:
		m.menuItems = m.getContainerMenuItems(node)
	}
}

func (m *Model) getProjectMenuItems(node *model.TreeNode) []MenuItem {
	// Capture the children slice to avoid closure issues
	children := node.Children
	
	return []MenuItem{
		{
			Label: "Restart All",
			Action: func() tea.Cmd {
				return func() tea.Msg {
					// Run in background
					go func() {
						for _, child := range children {
							if child.Container != nil && child.Container.State == "running" {
								m.dockerClient.RestartContainer(child.Container.ID)
							}
						}
					}()
					// Immediately refresh to show operation started
					return m.refreshContainers()()
				}
			},
		},
		{
			Label: "Stop All",
			Action: func() tea.Cmd {
				return func() tea.Msg {
					// Run in background
					go func() {
						for _, child := range children {
							if child.Container != nil && child.Container.State == "running" {
								m.dockerClient.StopContainer(child.Container.ID)
							}
						}
					}()
					// Immediately refresh to show operation started
					return m.refreshContainers()()
				}
			},
		},
		{
			Label: "Down (stop & remove, keeps volumes)",
			Action: func() tea.Cmd {
				return func() tea.Msg {
					// Run in background
					go func() {
						for _, child := range children {
							if child.Container != nil {
								// Stop and remove containers (volumes are preserved)
								m.dockerClient.RemoveContainer(child.Container.ID)
							}
						}
					}()
					// Immediately refresh to show operation started
					return m.refreshContainers()()
				}
			},
		},
		{
			Label: "Start All",
			Action: func() tea.Cmd {
				return func() tea.Msg {
					// Run in background
					go func() {
						for _, child := range children {
							if child.Container != nil && child.Container.State != "running" {
								m.dockerClient.StartContainer(child.Container.ID)
							}
						}
					}()
					// Immediately refresh to show operation started
					return m.refreshContainers()()
				}
			},
		},
	}
}

func (m *Model) getContainerMenuItems(node *model.TreeNode) []MenuItem {
	container := node.Container
	if container == nil {
		return []MenuItem{}
	}

	// Capture container ID to avoid closure issues
	containerID := container.ID
	containerState := container.State

	items := []MenuItem{}

	if containerState == "running" {
		items = append(items, MenuItem{
			Label: "Restart",
			Action: func() tea.Cmd {
				return func() tea.Msg {
					// Run in background
					go func() {
						m.dockerClient.RestartContainer(containerID)
					}()
					// Immediately refresh to show operation started
					return m.refreshContainers()()
				}
			},
		})
		items = append(items, MenuItem{
			Label: "Stop",
			Action: func() tea.Cmd {
				return func() tea.Msg {
					// Run in background
					go func() {
						m.dockerClient.StopContainer(containerID)
					}()
					// Immediately refresh to show operation started
					return m.refreshContainers()()
				}
			},
		})
		items = append(items, MenuItem{
			Label: "Remove (keeps volumes)",
			Action: func() tea.Cmd {
				return func() tea.Msg {
					// Run in background
					go func() {
						m.dockerClient.RemoveContainer(containerID)
					}()
					// Immediately refresh to show operation started
					return m.refreshContainers()()
				}
			},
		})
	} else {
		items = append(items, MenuItem{
			Label: "Start",
			Action: func() tea.Cmd {
				return func() tea.Msg {
					// Run in background
					go func() {
						m.dockerClient.StartContainer(containerID)
					}()
					// Immediately refresh to show operation started
					return m.refreshContainers()()
				}
			},
		})
	}

	items = append(items, MenuItem{
		Label: "Logs",
		Action: func() tea.Cmd {
			return func() tea.Msg {
				logs, err := m.dockerClient.GetContainerLogs(containerID, 1000)
				if err != nil {
					return errMsg{err}
				}
				return logsMsg{
					containerName: container.Name,
					content:       logs,
				}
			}
		},
	})

	// TODO: Add inspect when implemented
	// items = append(items, MenuItem{
	// 	Label:  "Inspect",
	// 	Action: func() tea.Cmd { return nil },
	// })

	return items
}

func (m Model) View() string {
	return m.renderView()
}

// adjustViewport ensures the selected item is visible in the viewport
func (m *Model) adjustViewport() {
	if m.tree == nil || len(m.tree.Flat) == 0 {
		return
	}

	// Calculate visible height (total - title/header - footer)
	// Title + blank line = 2, Header = 1, Footer + blank = 2, Total overhead = 5
	visibleHeight := m.height - 5
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	selected := m.tree.Selected

	// Scroll down if selected is below viewport
	if selected >= m.viewportTop+visibleHeight {
		m.viewportTop = selected - visibleHeight + 1
	}

	// Scroll up if selected is above viewport
	if selected < m.viewportTop {
		m.viewportTop = selected
	}

	// Ensure viewport doesn't go negative
	if m.viewportTop < 0 {
		m.viewportTop = 0
	}
}

