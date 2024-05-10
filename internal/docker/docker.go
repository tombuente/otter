package docker

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
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
	Containers []ContainerConfig `toml:"service"`
	Volumes    []VolumeConfig    `toml:"volume"`
}

type VolumeConfig struct {
	Name   string
	Labels []KeyValue
}

type ContainerConfig struct {
	Image   string
	Name    string
	Restart string
	Ports   []Port
	EnvVars []KeyValue
	Mounts  []Mount
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
	}

	return provider, nil
}

func (provider ProviderImpl) Up(ctx context.Context) error {
	slog.Info("Pulling Docker images")
	if err := provider.pullImages(ctx); err != nil {
		return fmt.Errorf("unable to pull Docker images: %w", err)
	}

	slog.Info("Creating Docker volumes")
	if err := provider.createVolumes(ctx); err != nil {
		return fmt.Errorf("unable to create Docker volumes: %w", err)
	}

	slog.Info("Creating Docker containers")
	responses, err := provider.createContainers(ctx)
	if err != nil {
		return fmt.Errorf("unable to create Docker containers: %w", err)
	}

	slog.Info("Starting Docker containers")
	if err := provider.startContainers(ctx, responses); err != nil {
		return fmt.Errorf("unable to start Docker containers: %w", err)
	}

	return nil
}

func (provider ProviderImpl) Close() {
	provider.client.Close()
}

func (provider ProviderImpl) pullImages(ctx context.Context) error {
	for _, c := range provider.config.Containers {
		reader, err := provider.client.ImagePull(ctx, c.Image, image.PullOptions{})
		if err != nil {
			return fmt.Errorf("unable to pull Docker image: %w", err)
		}
		defer reader.Close()
		io.Copy(io.Discard, reader)
	}

	return nil
}

func (provider ProviderImpl) createVolumes(ctx context.Context) error {
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

	return nil
}

func (provider ProviderImpl) createContainers(ctx context.Context) ([]container.CreateResponse, error) {
	var responses []container.CreateResponse

	for _, c := range provider.config.Containers {
		containerConfig := newContainerConfig(c)

		hostConfig, err := newHostConfig(c)
		if err != nil {
			return []container.CreateResponse{}, fmt.Errorf("unable to create host config: %w", err)
		}

		slog.Info("Creating Docker container")
		createRes, err := provider.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, c.Name)
		if err != nil {
			return []container.CreateResponse{}, fmt.Errorf("unable to create Docker container: %w", err)
		}

		responses = append(responses, createRes)
	}

	return responses, nil
}

func (provider ProviderImpl) startContainers(ctx context.Context, responses []container.CreateResponse) error {
	for _, res := range responses {
		if err := provider.client.ContainerStart(ctx, res.ID, container.StartOptions{}); err != nil {
			return err
		}
	}

	return nil
}
