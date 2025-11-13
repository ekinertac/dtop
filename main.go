package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ekinertac/dtop/docker"
	"github.com/ekinertac/dtop/ui"
)

func main() {
	ctx := context.Background()

	// Initialize Docker client
	dockerClient, err := docker.NewClient(ctx)
	if err != nil {
		fmt.Printf("Failed to create Docker client: %v\n", err)
		os.Exit(1)
	}
	defer dockerClient.Close()

	// Initialize bubbletea model
	model := ui.NewModel(dockerClient)

	// Start the program
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
