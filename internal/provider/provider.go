package provider

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/pelletier/go-toml/v2"
	"github.com/tombuente/otter/internal/docker"
)

const (
	DockerProvider ProviderKind = "docker"
)

type Provider interface {
	Up(ctx context.Context) error
	Close()
}

type Config struct {
	Provider string
}

type ProviderKind string

func NewProvider(config []byte) (Provider, error) {
	decoder := toml.NewDecoder(bytes.NewReader(config))
	genericConfig := Config{}
	err := decoder.Decode(&genericConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to get provider name from file: %w", err)
	}
	kind := ProviderKind(genericConfig.Provider)

	decoder = toml.NewDecoder(bytes.NewReader(config))
	switch kind {
	case DockerProvider:
		config := docker.Config{}
		err := decoder.Decode(&config)
		if err != nil {
			return nil, fmt.Errorf("unable to parse config for Docker provider: %w", err)
		}

		provider, err := docker.NewProvider(config)
		if err != nil {
			return nil, fmt.Errorf("unable to create Docker provider: %w", err)
		}

		return provider, nil
	default:
		return nil, errors.New("provider not supported")
	}
}
