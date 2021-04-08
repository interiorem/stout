package docker

import (
	"github.com/docker/engine-api/types/container"
	"github.com/interiorem/stout/isolate"
	"github.com/tinylib/msgp/msgp"
)

const (
	defaultRuntimePath  = "/var/run/cocaine"
	defatultNetworkMode = container.NetworkMode("bridge")
)

//go:generate msgp -o profile_decodable.go

type Resources struct {
	Memory     msgp.Number `msg:"memory"`
	CPUShares  msgp.Number `msg:"CpuShares"`
	CPUPeriod  msgp.Number `msg:"CpuPeriod"` // CPU CFS (Completely Fair Scheduler) period
	CPUQuota   msgp.Number `msg:"CpuQuota"`  // CPU CFS (Completely Fair Scheduler) quota
	CpusetCpus string      `msg:"CpusetCpus"`
	CpusetMems string      `msg:"CpusetMems"`
}

// Profile describes a Cocaine profile for Docker isolation type
type Profile struct {
	Registry   string `msg:"registry"`
	Repository string `msg:"repository"`
	Endpoint   string `msg:"endpoint"`

	NetworkMode string `msg:"network_mode"`
	RuntimePath string `msg:"runtime-path"`
	Cwd         string `msg:"cwd"`

	Resources `msg:"resources"`
	Tmpfs     map[string]string `msg:"tmpfs"`
	Binds     []string          `msg:"binds"`
}

func decodeProfile(raw isolate.RawProfile) (*Profile, error) {
	profile := Profile{
		NetworkMode: "bridge",
		RuntimePath: defaultRuntimePath,
	}

	err := raw.DecodeTo(&profile)
	return &profile, err
}
