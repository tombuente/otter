package docker

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
)

func newHostConfig(config ContainerConfig) (*container.HostConfig, error) {
	portBindings := hostPortBindings(config.Ports)

	restartPolicy, err := hostRestartPolicy(config.Restart)
	if err != nil {
		return &container.HostConfig{}, err
	}

	mounts := hostMounts(config.Mounts)

	return &container.HostConfig{
		PortBindings:  portBindings,
		RestartPolicy: restartPolicy,
		Mounts:        mounts,
	}, nil
}

func hostRestartPolicy(restart string) (container.RestartPolicy, error) {
	var hostRestartPolicy container.RestartPolicy
	switch restart {
	case "no":
		hostRestartPolicy.Name = container.RestartPolicyDisabled
	case "always":
		hostRestartPolicy.Name = container.RestartPolicyAlways
	case "on-failure":
		hostRestartPolicy.Name = container.RestartPolicyOnFailure
	case "", "unless-stopped":
		hostRestartPolicy.Name = container.RestartPolicyUnlessStopped
	default:
		return container.RestartPolicy{}, fmt.Errorf("restart policy not supported: %s", restart)
	}

	return hostRestartPolicy, nil
}

func hostPortBindings(ports []Port) nat.PortMap {
	portBindings := make(nat.PortMap)
	for _, port := range ports {
		portBindings[nat.Port(fmt.Sprintf("%v/%v", port.Host, port.Protocol))] = []nat.PortBinding{
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

	return portBindings
}

func hostMounts(mounts []Mount) []mount.Mount {
	var hostMounts []mount.Mount
	for _, m := range mounts {
		mountType := mount.Type(m.Type)
		if mountType == "" {
			mountType = mount.Type("bind")
		}

		newMount := mount.Mount{
			Source: m.Host,
			Target: m.Container,
			Type:   mountType,
		}

		hostMounts = append(hostMounts, newMount)
	}

	return hostMounts
}
