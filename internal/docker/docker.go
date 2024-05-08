package docker

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

type ProviderImpl struct {
	client *client.Client
	config Config
}

type KeyValue struct {
	Name  string
	Value string
}

type Config struct {
	Services []ServiceConfig `toml:"service"`
	Volumes  []VolumeConfig  `toml:"volume"`
}

type VolumeConfig struct {
	Name   string
	Labels []KeyValue
}

type ServiceConfig struct {
	Image   string
	Name    string
	Restart string
	Ports   []ServiceConfigPorts
	Env     []KeyValue
	Mounts  []ServiceConfigMounts
}

type ServiceConfigPorts struct {
	Host      string
	Container string
	Protocol  string
}

type ServiceConfigMounts struct {
	Host      string
	Container string
	Type      string
}

func NewProvider(config Config) (ProviderImpl, error) {
	client, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return ProviderImpl{}, nil
	}

	provider := ProviderImpl{
		client: client,
		config: config,
	}

	return provider, nil
}

func ports(ports []ServiceConfigPorts) (nat.PortSet, nat.PortMap) {
	containerExposedPorts := make(nat.PortSet)
	hostPortBindings := make(nat.PortMap)
	for _, port := range ports {
		containerExposedPorts[nat.Port(fmt.Sprintf("%v/%v", port.Container, port.Protocol))] = struct{}{}
		hostPortBindings[nat.Port(fmt.Sprintf("%v/%v", port.Host, port.Protocol))] = []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: port.Host,
			},
			{
				HostIP:   "::",
				HostPort: port.Host,
			},
		}
	}

	return containerExposedPorts, hostPortBindings
}

func (provider ProviderImpl) Up(ctx context.Context) error {
	for _, v := range provider.config.Volumes {
		labels := make(map[string]string)
		for _, label := range v.Labels {
			labels[label.Name] = label.Value
		}

		options := volume.CreateOptions{
			Name:   v.Name,
			Labels: labels,
		}

		_, err := provider.client.VolumeCreate(ctx, options)
		if err != nil {
			return fmt.Errorf("unable to create Docker volume: %w", err)
		}
	}

	for _, service := range provider.config.Services {
		reader, err := provider.client.ImagePull(ctx, service.Image, image.PullOptions{})
		if err != nil {
			return err
		}
		defer reader.Close()
		io.Copy(io.Discard, reader) // The reader needs to be read completely for the pull operation to complete.

		containerExposedPorts, hostPortBindings := ports(service.Ports)

		var containerEnv []string
		for _, envVar := range service.Env {
			containerEnv = append(containerEnv, fmt.Sprintf("%v=%v", envVar.Name, envVar.Value))
		}

		var hostRestartPolicy container.RestartPolicy
		switch service.Restart {
		case "no":
			hostRestartPolicy.Name = container.RestartPolicyDisabled
		case "always":
			hostRestartPolicy.Name = container.RestartPolicyAlways
		case "on-failure":
			hostRestartPolicy.Name = container.RestartPolicyOnFailure
		case "unless-stopped":
			hostRestartPolicy.Name = container.RestartPolicyUnlessStopped
		}

		var hostMounts []mount.Mount
		for _, m := range service.Mounts {
			newMount := mount.Mount{
				Source: m.Host,
				Target: m.Container,
				Type:   mount.Type(m.Type),
			}

			hostMounts = append(hostMounts, newMount)
		}

		containerConfig := &container.Config{
			Image:        service.Image,
			ExposedPorts: containerExposedPorts,
			Env:          containerEnv,
		}

		hostConfig := &container.HostConfig{
			PortBindings:  hostPortBindings,
			RestartPolicy: hostRestartPolicy,
			Mounts:        hostMounts,
		}

		createRes, err := provider.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, service.Name)
		if err != nil {
			return fmt.Errorf("unable to create Docker container: %w", err)
		}

		if err := provider.client.ContainerStart(ctx, createRes.ID, container.StartOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func (provider ProviderImpl) Close() {
	provider.client.Close()
}
