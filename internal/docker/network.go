package docker

import (
	"errors"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
)

func newNetworkConfig(config ContainerConfig, networks map[string]types.NetworkCreateResponse) (network.NetworkingConfig, error) {
	settings := make(map[string]*network.EndpointSettings)
	for _, name := range config.Networks {
		networkRes, ok := networks[name]
		if !ok {
			return network.NetworkingConfig{}, errors.New("network not in network map")
		}

		settings[name] = &network.EndpointSettings{
			NetworkID: networkRes.ID,
		}
	}

	return network.NetworkingConfig{
		EndpointsConfig: settings,
	}, nil
}
