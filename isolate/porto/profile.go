package porto

import (
	"github.com/mitchellh/mapstructure"

	"github.com/noxiouz/stout/isolate"
)

const (
	defaultRuntimePath = "/var/run/cocaine"
)

type volumeProfile struct {
	Target     string            `json:"target"`
	Properties map[string]string `json:"properties"`
}

type Profile struct {
	Registry   string `json:"registry"`
	Repository string `json:"repository"`

	NetworkMode string `json:"network_mode"`
	Cwd         string `json:"cwd"`

	Binds []string `json:"binds"`

	Container    map[string]string `json:"container"`
	Volume       map[string]string `jsonL:"volume"`
	ExtraVolumes []volumeProfile   `json:"extravolumes"`
}

// ConvertProfile unpacked general profile to a Docker specific
func ConvertProfile(rawprofile isolate.Profile) (*Profile, error) {
	var profile = &Profile{}

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
