package docker

import (
	"github.com/docker/engine-api/types/container"
	"github.com/mitchellh/mapstructure"

	"github.com/noxiouz/stout/isolate"
)

const (
	defaultRuntimePath  = "/var/run/cocaine"
	defatultNetworkMode = container.NetworkMode("bridge")
)

// Profile describes a Cocaine profile for Docker isolation type
type Profile struct {
	Registry   string `json:"registry"`
	Repository string `json:"repository"`
	Endpoint   string `json:"endpoint"`

	NetworkMode container.NetworkMode `json:"network_mode"`
	RuntimePath string                `json:"runtime-path"`
	Cwd         string                `json:"cwd"`
}

func convertProfile(rawprofile isolate.Profile) (*Profile, error) {
	// Create profile with default values
	// They can be overwritten by decode
	var profile = &Profile{
		NetworkMode: container.NetworkMode("bridge"),
		RuntimePath: defaultRuntimePath,
	}

	config := mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           profile,
		TagName:          "json",
	}

	decoder, err := mapstructure.NewDecoder(&config)
	if err != nil {
		return nil, err
	}

	return profile, decoder.Decode(rawprofile)
}
