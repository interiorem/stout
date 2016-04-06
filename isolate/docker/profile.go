package docker

import (
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types/container"

	"github.com/noxiouz/stout/isolate"
)

const (
	defaultRuntimePath  = "/var/run/cocaine"
	defatultNetworkMode = container.NetworkMode("bridge")
)

// TODO: use mapstructurer with metatags

type Profile isolate.Profile

func (p Profile) Endpoint() string {
	if endpoint, ok := p["endpoint"].(string); ok {
		return endpoint
	}

	return client.DefaultDockerHost
}

func (p Profile) Registry() string {
	if registry, ok := p["registry"].(string); ok {
		return registry
	}

	return ""
}

func (p Profile) NetworkMode() container.NetworkMode {
	if endpoint, ok := p["network_mode"].(string); ok {
		return container.NetworkMode(endpoint)
	}

	return defatultNetworkMode
}

func (p Profile) RuntimePath() string {
	if runtimepath, ok := p["runtime-path"].(string); ok {
		return runtimepath
	}

	return defaultRuntimePath
}
