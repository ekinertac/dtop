package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ekinertac/dtop/docker"
	"github.com/ekinertac/dtop/model"
)

type Model struct {
	dockerClient *docker.Client
	tree         *model.Tree
	menuOpen     bool
	menuItems    []MenuItem
	menuSelected int
	width        int
	height       int
	viewportTop  int // First visible line in the tree
	err          error
}

type MenuItem struct {
	Label  string
	Action func() tea.Cmd
}

type tickMsg time.Time

func NewModel(dockerClient *docker.Client) Model {
	return Model{
		dockerClient: dockerClient,
		tree:         &model.Tree{},
		menuOpen:     false,
		menuSelected: 0,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.refreshContainers(),
		tickCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) refreshContainers() tea.Cmd {
	return func() tea.Msg {
		containers, err := m.dockerClient.ListContainers()
		if err != nil {
			return errMsg{err}
		}
		return containersMsg(containers)
	}
}

type containersMsg []docker.ContainerInfo
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
		// Preserve selection across refresh
		var selectedPath string
		if m.tree != nil {
			selectedNode := m.tree.GetSelected()
			if selectedNode != nil {
				selectedPath = m.tree.GetNodePath(selectedNode)
			}
		}
		
		m.tree = model.BuildTree(msg)
		
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

	case errMsg:
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	}

	return m, nil
}

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle menu navigation
	if m.menuOpen {
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
				m.menuOpen = false
				return m, cmd
			}
		case "esc":
			m.menuOpen = false
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
	m.menuOpen = true

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
			Label: "Stop & Remove All (keeps volumes)",
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
			Label: "Stop & Remove (keeps volumes)",
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

	// TODO: Add these when implemented
	// items = append(items, MenuItem{
	// 	Label:  "View Logs",
	// 	Action: func() tea.Cmd { return nil },
	// })
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

