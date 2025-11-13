package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	NetRx     uint64 // Network bytes received
	NetTx     uint64 // Network bytes transmitted
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
	return c.ListContainersWithStats(true)
}

func (c *Client) ListContainersWithStats(includeStats bool) ([]ContainerInfo, error) {
	// Only list running containers (equivalent to `docker ps` without -a)
	containers, err := c.cli.ContainerList(c.ctx, container.ListOptions{All: false})
	if err != nil {
		return nil, err
	}

	// Build initial result without stats
	result := make([]ContainerInfo, len(containers))
	type statsResult struct {
		index    int
		cpuPerc  float64
		memPerc  float64
		memUsage string
		netRx    uint64
		netTx    uint64
	}
	statsChan := make(chan statsResult, len(containers))

	// Fetch stats in parallel for running containers
	runningCount := 0
	for i, ctr := range containers {
		name := strings.TrimPrefix(ctr.Names[0], "/")

		result[i] = ContainerInfo{
			ID:        ctr.ID[:12],
			Name:      name,
			Image:     ctr.Image,
			State:     ctr.State,
			Status:    ctr.Status,
			CPUPerc:   0.0,
			MemPerc:   0.0,
			MemUsage:  "N/A",
			NetRx:     0,
			NetTx:     0,
			CreatedAt: time.Unix(ctr.Created, 0),
			Labels:    ctr.Labels,
		}

		if ctr.State == "running" && includeStats {
			runningCount++
			go func(idx int, containerID string) {
				stats := c.getContainerStats(containerID)
				statsChan <- statsResult{
					index:    idx,
					cpuPerc:  stats.cpuPerc,
					memPerc:  stats.memPerc,
					memUsage: stats.memUsage,
					netRx:    stats.netRx,
					netTx:    stats.netTx,
				}
			}(i, ctr.ID)
		}
	}

	// Collect stats results (only if requested)
	if includeStats {
		for i := 0; i < runningCount; i++ {
			stats := <-statsChan
			result[stats.index].CPUPerc = stats.cpuPerc
			result[stats.index].MemPerc = stats.memPerc
			result[stats.index].MemUsage = stats.memUsage
			result[stats.index].NetRx = stats.netRx
			result[stats.index].NetTx = stats.netTx
		}
	}

	return result, nil
}

// Stats structures for parsing Docker stats JSON
type statsResponse struct {
	CPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemUsage uint64 `json:"system_cpu_usage"`
		OnlineCPUs  uint32 `json:"online_cpus"`
	} `json:"cpu_stats"`
	PreCPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemUsage uint64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Usage uint64 `json:"usage"`
		Limit uint64 `json:"limit"`
	} `json:"memory_stats"`
	Networks map[string]struct {
		RxBytes uint64 `json:"rx_bytes"`
		TxBytes uint64 `json:"tx_bytes"`
	} `json:"networks"`
}

type statsData struct {
	cpuPerc  float64
	memPerc  float64
	memUsage string
	netRx    uint64
	netTx    uint64
}

func (c *Client) getContainerStats(containerID string) statsData {
	// Get a single stats snapshot (stream=false)
	stats, err := c.cli.ContainerStats(c.ctx, containerID, false)
	if err != nil {
		return statsData{0.0, 0.0, "N/A", 0, 0}
	}
	defer stats.Body.Close()

	// Decode the stats
	var v statsResponse
	if err := json.NewDecoder(stats.Body).Decode(&v); err != nil && err != io.EOF {
		return statsData{0.0, 0.0, "N/A", 0, 0}
	}

	result := statsData{}

	// Calculate CPU percentage
	cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage - v.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(v.CPUStats.SystemUsage - v.PreCPUStats.SystemUsage)
	onlineCPUs := float64(v.CPUStats.OnlineCPUs)
	
	if systemDelta > 0.0 && cpuDelta > 0.0 {
		result.cpuPerc = (cpuDelta / systemDelta) * onlineCPUs * 100.0
	}

	// Calculate memory percentage
	if v.MemoryStats.Limit > 0 {
		result.memPerc = (float64(v.MemoryStats.Usage) / float64(v.MemoryStats.Limit)) * 100.0
	}

	// Format memory usage
	result.memUsage = formatBytes(v.MemoryStats.Usage) + " / " + formatBytes(v.MemoryStats.Limit)

	// Calculate network totals across all interfaces
	for _, net := range v.Networks {
		result.netRx += net.RxBytes
		result.netTx += net.TxBytes
	}

	return result
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return "0 B"
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
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

func (c *Client) GetContainerLogs(containerID string, tail int) (string, error) {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       fmt.Sprintf("%d", tail),
	}

	logs, err := c.cli.ContainerLogs(c.ctx, containerID, options)
	if err != nil {
		return "", err
	}
	defer logs.Close()

	// Read all logs
	buf := make([]byte, 1024*1024) // 1MB buffer
	n, _ := logs.Read(buf)

	return string(buf[:n]), nil
}
