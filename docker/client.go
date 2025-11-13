package docker

import (
	"context"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type Client struct {
	cli *client.Client
	ctx context.Context
}

type ContainerInfo struct {
	ID        string
	Name      string
	Image     string
	State     string
	Status    string
	CPUPerc   float64
	MemPerc   float64
	MemUsage  string
	NetIO     string
	BlockIO   string
	CreatedAt time.Time
	Labels    map[string]string
}

func NewClient(ctx context.Context) (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return &Client{
		cli: cli,
		ctx: ctx,
	}, nil
}

func (c *Client) Close() error {
	return c.cli.Close()
}

func (c *Client) ListContainers() ([]ContainerInfo, error) {
	// Only list running containers (equivalent to `docker ps` without -a)
	containers, err := c.cli.ContainerList(c.ctx, container.ListOptions{All: false})
	if err != nil {
		return nil, err
	}

	var result []ContainerInfo
	for _, ctr := range containers {
		// Get container stats for CPU and memory
		// Note: For now, we'll use placeholder values for stats
		// In the future, we can implement real-time stats collection
		var cpuPerc, memPerc float64 = 0.0, 0.0
		var memUsage string = "N/A"

		name := ctr.Names[0]
		if strings.HasPrefix(name, "/") {
			name = name[1:]
		}

		result = append(result, ContainerInfo{
			ID:        ctr.ID[:12],
			Name:      name,
			Image:     ctr.Image,
			State:     ctr.State,
			Status:    ctr.Status,
			CPUPerc:   cpuPerc,
			MemPerc:   memPerc,
			MemUsage:  memUsage,
			CreatedAt: time.Unix(ctr.Created, 0),
			Labels:    ctr.Labels,
		})
	}

	return result, nil
}

func (c *Client) RestartContainer(containerID string) error {
	timeout := 10
	return c.cli.ContainerRestart(c.ctx, containerID, container.StopOptions{Timeout: &timeout})
}

func (c *Client) StopContainer(containerID string) error {
	timeout := 10
	return c.cli.ContainerStop(c.ctx, containerID, container.StopOptions{Timeout: &timeout})
}

func (c *Client) StartContainer(containerID string) error {
	return c.cli.ContainerStart(c.ctx, containerID, container.StartOptions{})
}

func (c *Client) RemoveContainer(containerID string) error {
	return c.cli.ContainerRemove(c.ctx, containerID, container.RemoveOptions{
		Force:         true,  // Force removal even if running
		RemoveVolumes: false, // Keep volumes (preserve data)
	})
}

