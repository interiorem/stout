package docker

import (
	"github.com/docker/engine-api/client"

	"github.com/noxiouz/stout/isolation"
)

type Profile isolation.Profile

func (p Profile) Endpoint() string {
	if endpoint, ok := p["endpoint"].(string); ok {
		return endpoint
	}

	return client.DefaultDockerHost
}
