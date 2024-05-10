package docker

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
)

func newContainerConfig(config ContainerConfig) *container.Config {
	exposedPorts := containerExposedPorts(config.Ports)
	env := containerEnv(config.EnvVars)

	return &container.Config{
		Image:        config.Image,
		ExposedPorts: exposedPorts,
		Env:          env,
	}
}

func containerExposedPorts(ports []Port) nat.PortSet {
	exposedPorts := make(nat.PortSet)
	for _, port := range ports {
		exposedPorts[nat.Port(fmt.Sprintf("%v/%v", port.Container, port.Protocol))] = struct{}{}
	}

	return exposedPorts
}

func containerEnv(envVars []KeyValue) []string {
	var env []string
	for _, envVar := range envVars {
		env = append(env, fmt.Sprintf("%v=%v", envVar.Name, envVar.Value))
	}

	return env
}
