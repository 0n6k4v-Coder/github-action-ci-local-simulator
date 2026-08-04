package dockerx

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// ServiceConfig represents container configuration for a service.
type ServiceConfig struct {
	Name    string
	Image   string
	Env     map[string]any
	Ports   []any
	Options string
}

// CreateNetwork creates a Docker bridge network with the specified name.
func CreateNetwork(ctx context.Context, cli *client.Client, networkName string) (string, error) {
	if cli == nil {
		return "mock-network-id", nil
	}
	resp, err := cli.NetworkCreate(ctx, networkName, network.CreateOptions{
		Driver: "bridge",
	})
	if err != nil {
		return "", fmt.Errorf("create docker network: %w", err)
	}
	return resp.ID, nil
}

// RemoveNetwork removes a Docker network by ID.
func RemoveNetwork(ctx context.Context, cli *client.Client, networkID string) error {
	if cli == nil || networkID == "mock-network-id" {
		return nil
	}
	if err := cli.NetworkRemove(ctx, networkID); err != nil {
		return fmt.Errorf("remove docker network: %w", err)
	}
	return nil
}

// ConnectNetwork connects a container to a Docker network.
func ConnectNetwork(ctx context.Context, cli *client.Client, networkID, containerID string) error {
	if cli == nil || networkID == "mock-network-id" || strings.HasPrefix(containerID, "mock-") {
		return nil
	}
	if err := cli.NetworkConnect(ctx, networkID, containerID, nil); err != nil {
		return fmt.Errorf("connect container to network: %w", err)
	}
	return nil
}

// CreateServiceContainer creates a service container connected to the specified network.
func CreateServiceContainer(ctx context.Context, cli *client.Client, networkName, serviceName string, serviceConfig ServiceConfig) (string, string, error) {
	// Parse environment variables
	var envVars []string
	for k, v := range serviceConfig.Env {
		envVars = append(envVars, fmt.Sprintf("%s=%v", k, v))
	}

	// Parse ports
	exposedPorts, portBindings, primaryPort, err := ParsePorts(serviceConfig.Ports)
	if err != nil {
		return "", "", fmt.Errorf("parse ports for service %s: %w", serviceName, err)
	}

	// Parse health check from options
	healthConfig := ParseHealthConfig(serviceConfig.Options)

	if cli == nil {
		return fmt.Sprintf("mock-service-%s", serviceName), primaryPort, nil
	}

	// Networking config: set network alias to service name
	endpointsConfig := map[string]*network.EndpointSettings{
		networkName: {
			Aliases: []string{serviceName},
		},
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:        serviceConfig.Image,
		Hostname:     serviceName,
		Env:          envVars,
		ExposedPorts: exposedPorts,
		Healthcheck:  healthConfig,
	}, &container.HostConfig{
		AutoRemove:   false,
		PortBindings: portBindings,
	}, &network.NetworkingConfig{
		EndpointsConfig: endpointsConfig,
	}, nil, "")
	if err != nil {
		return "", "", fmt.Errorf("create service container %s: %w", serviceName, err)
	}

	return resp.ID, primaryPort, nil
}

// ParsePorts parses port mapping options into nat.PortSet and nat.PortMap.
func ParsePorts(ports []any) (nat.PortSet, nat.PortMap, string, error) {
	exposedPorts := make(nat.PortSet)
	portBindings := make(nat.PortMap)
	var primaryPort string

	for _, p := range ports {
		portStr := strings.TrimSpace(fmt.Sprintf("%v", p))
		if portStr == "" {
			continue
		}
		parts := strings.Split(portStr, ":")
		var hostPort, containerPort string
		if len(parts) == 2 {
			hostPort = strings.TrimSpace(parts[0])
			containerPort = strings.TrimSpace(parts[1])
		} else if len(parts) == 1 {
			hostPort = strings.TrimSpace(parts[0])
			containerPort = strings.TrimSpace(parts[0])
		} else {
			return nil, nil, "", fmt.Errorf("invalid port format: %s", portStr)
		}

		if primaryPort == "" {
			primaryPort = hostPort
		}

		cPort, err := nat.NewPort("tcp", containerPort)
		if err != nil {
			return nil, nil, "", fmt.Errorf("invalid container port %s: %w", containerPort, err)
		}

		exposedPorts[cPort] = struct{}{}
		portBindings[cPort] = []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: hostPort,
			},
		}
	}

	return exposedPorts, portBindings, primaryPort, nil
}

// ParseHealthConfig parses health check options from the options string.
func ParseHealthConfig(optionsStr string) *container.HealthConfig {
	optionsStr = strings.TrimSpace(optionsStr)
	if optionsStr == "" {
		return nil
	}

	args := splitOptionsArgs(optionsStr)
	if len(args) == 0 {
		return nil
	}

	hc := &container.HealthConfig{}
	hasHealthcheck := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		var key, val string

		if strings.HasPrefix(arg, "--") {
			parts := strings.SplitN(arg, "=", 2)
			key = parts[0]
			if len(parts) == 2 {
				val = parts[1]
			} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				val = args[i+1]
				i++
			}
		}

		switch key {
		case "--health-cmd":
			if val != "" {
				hc.Test = []string{"CMD-SHELL", val}
				hasHealthcheck = true
			}
		case "--health-interval":
			if d, err := time.ParseDuration(val); err == nil {
				hc.Interval = d
				hasHealthcheck = true
			}
		case "--health-timeout":
			if d, err := time.ParseDuration(val); err == nil {
				hc.Timeout = d
				hasHealthcheck = true
			}
		case "--health-start-period":
			if d, err := time.ParseDuration(val); err == nil {
				hc.StartPeriod = d
				hasHealthcheck = true
			}
		case "--health-retries":
			if r, err := strconv.Atoi(val); err == nil {
				hc.Retries = r
				hasHealthcheck = true
			}
		}
	}

	if !hasHealthcheck {
		return nil
	}
	return hc
}

func splitOptionsArgs(s string) []string {
	var args []string
	var current strings.Builder
	inQuotes := false
	var quoteChar rune

	for _, r := range s {
		switch {
		case (r == '"' || r == '\'') && !inQuotes:
			inQuotes = true
			quoteChar = r
		case r == quoteChar && inQuotes:
			inQuotes = false
			quoteChar = 0
		case (r == ' ' || r == '\t' || r == '\n') && !inQuotes:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

// WaitForServiceReady waits for a service container to become ready.
func WaitForServiceReady(ctx context.Context, cli *client.Client, containerID string, hostPort string, maxWait time.Duration) error {
	if cli == nil || strings.HasPrefix(containerID, "mock-") {
		return nil
	}

	if maxWait <= 0 {
		maxWait = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, maxWait)
	defer cancel()

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for service container %s to be ready", containerID)
		case <-ticker.C:
			inspect, err := cli.ContainerInspect(ctx, containerID)
			if err != nil {
				return fmt.Errorf("inspect container: %w", err)
			}

			if inspect.State == nil {
				continue
			}

			if !inspect.State.Running {
				return fmt.Errorf("service container stopped unexpectedly with exit code %d", inspect.State.ExitCode)
			}

			// If Docker health check status is available
			if inspect.State.Health != nil {
				switch inspect.State.Health.Status {
				case "healthy":
					return nil
				case "unhealthy":
					return fmt.Errorf("service container is unhealthy")
				}
				// If "starting", continue waiting
				continue
			}

			// If TCP port is mapped, try connecting
			if hostPort != "" {
				conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", hostPort), 250*time.Millisecond)
				if err == nil {
					_ = conn.Close()
					return nil
				}
				continue
			}

			// If container is running and no port or healthcheck was set, it's ready
			return nil
		}
	}
}
