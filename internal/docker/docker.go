package docker

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
)

type ProviderImpl struct {
	client *client.Client
	config Config

	containers map[string]container.CreateResponse
	networks   map[string]types.NetworkCreateResponse
}

type KeyValue struct {
	Name  string
	Value string
}

type Config struct {
	Containers map[string]ContainerConfig `toml:"service"`
	Networks   map[string]NetworkConfig   `toml:"network"`
	Volumes    map[string]VolumeConfig    `toml:"volume"`
}

type ContainerConfig struct {
	Image    string
	Restart  string
	Ports    []Port
	Networks []string
	EnvVars  []KeyValue
	Mounts   []Mount
}

type NetworkConfig struct{}

type VolumeConfig struct {
	Labels []KeyValue
}

type Port struct {
	Host      string
	Container string
	Protocol  string
}

type Mount struct {
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

		containers: make(map[string]container.CreateResponse),
		networks:   make(map[string]types.NetworkCreateResponse),
	}

	return provider, nil
}

func (provider ProviderImpl) Up(ctx context.Context) error {
	slog.Info("Pulling Docker images")
	if err := provider.pullImages(ctx); err != nil {
		return fmt.Errorf("unable to pull Docker images: %w", err)
	}

	slog.Info("Creating Docker networks")
	if err := provider.createNetworks(ctx); err != nil {
		return fmt.Errorf("unable to create Docker networks: %w", err)
	}

	slog.Info("Creating Docker volumes")
	if err := provider.createVolumes(ctx); err != nil {
		return fmt.Errorf("unable to create Docker volumes: %w", err)
	}

	slog.Info("Creating Docker containers")
	err := provider.createContainers(ctx)
	if err != nil {
		return fmt.Errorf("unable to create Docker containers: %w", err)
	}

	slog.Info("Starting Docker containers")
	if err := provider.startContainers(ctx); err != nil {
		return fmt.Errorf("unable to start Docker containers: %w", err)
	}

	return nil
}

func (provider ProviderImpl) Close() {
	provider.client.Close()
}

func (provider ProviderImpl) pullImages(ctx context.Context) error {
	for _, config := range provider.config.Containers {
		reader, err := provider.client.ImagePull(ctx, config.Image, image.PullOptions{})
		if err != nil {
			return fmt.Errorf("unable to pull Docker image: %w", err)
		}
		defer reader.Close()
		io.Copy(io.Discard, reader)
	}

	return nil
}

func (provider ProviderImpl) createNetworks(ctx context.Context) error {
	for name := range provider.config.Networks {
		options := types.NetworkCreate{}

		slog.Info("Creating Docker network", "name", name)
		networkRes, err := provider.client.NetworkCreate(ctx, name, options)
		switch {
		case errdefs.IsConflict(err):
			slog.Warn("Docker network already exists", "name", name)
		case err != nil:
			return fmt.Errorf("unable to create Docker network: %w", err)
		}

		provider.networks[name] = networkRes
	}

	return nil
}

func (provider ProviderImpl) createVolumes(ctx context.Context) error {
	for name, config := range provider.config.Volumes {
		labels := make(map[string]string)
		for _, label := range config.Labels {
			labels[label.Name] = label.Value
		}

		options := volume.CreateOptions{
			Name:   name,
			Labels: labels,
		}

		slog.Info("Creating Docker volume", "name", name)
		_, err := provider.client.VolumeCreate(ctx, options)
		if err != nil {
			return fmt.Errorf("unable to create Docker volume: %w", err)
		}
	}

	return nil
}

func (provider ProviderImpl) createContainers(ctx context.Context) error {
	for name, config := range provider.config.Containers {
		containerConfig := newContainerConfig(config)

		hostConfig, err := newHostConfig(config)
		if err != nil {
			return fmt.Errorf("unable to create host config: %w", err)
		}

		networkConfig, err := newNetworkConfig(config, provider.networks)
		if err != nil {
			return fmt.Errorf("unable to create network config: %w", err)
		}

		slog.Info("Creating Docker container", "name", name)
		createRes, err := provider.client.ContainerCreate(ctx, &containerConfig, &hostConfig, &networkConfig, nil, name)
		if err != nil {
			return fmt.Errorf("unable to create Docker container %v: %w", name, err)
		}

		provider.containers[name] = createRes
	}

	return nil
}

func (provider ProviderImpl) startContainers(ctx context.Context) error {
	for _, createRes := range provider.containers {
		if err := provider.client.ContainerStart(ctx, createRes.ID, container.StartOptions{}); err != nil {
			return err
		}
	}

	return nil
}
