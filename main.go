package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ekinertac/dtop/docker"
	"github.com/ekinertac/dtop/model"
	"github.com/ekinertac/dtop/ui"
)

func main() {
	// Parse command-line flags
	list := flag.Bool("list", false, "List containers and exit (non-interactive)")
	listShort := flag.Bool("l", false, "List containers and exit (shorthand)")
	version := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	// Version flag
	if *version {
		fmt.Println("dtop v0.1.0")
		fmt.Println("Docker container monitor - https://github.com/ekinertac/dtop")
		return
	}

	ctx := context.Background()

	// Initialize Docker client
	dockerClient, err := docker.NewClient(ctx)
	if err != nil {
		fmt.Printf("Failed to create Docker client: %v\n", err)
		os.Exit(1)
	}
	defer dockerClient.Close()

	// List mode - print once and exit
	if *list || *listShort {
		containers, err := dockerClient.ListContainers()
		if err != nil {
			fmt.Printf("Failed to list containers: %v\n", err)
			os.Exit(1)
		}

		tree := model.BuildTree(containers)
		ui.PrintSnapshot(tree)
		return
	}

	// Interactive mode - start TUI
	m := ui.NewModel(dockerClient)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
