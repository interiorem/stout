package porto

const (
	defaultRuntimePath = "/var/run/cocaine"
)

//go:generate msgp -o profile_decodable.go

type VolumeProfile struct {
	Target     string            `msg:"target"`
	Properties map[string]string `msg:"properties"`
}

type ExtendedInfo struct {
	Layers	map[string][]string
}

type Profile struct {
	Registry   string `msg:"registry"`
	Repository string `msg:"repository"`

	NetworkMode string `msg:"network_mode"`
	Network map[string]string `msg:"network"`

	ExtendedInfo ExtendedInfo `msg:"extended_info"`

	Cwd         string `msg:"cwd"`

	Binds []string `msg:"binds"`

	Container    map[string]string `msg:"container"`
	Volume       map[string]string `msg:"volume"`
	ExtraVolumes []VolumeProfile   `msg:"extravolumes"`
}
