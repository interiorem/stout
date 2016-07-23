package docker

import "github.com/noxiouz/stout/isolate"

func init() {
	isolate.RegisterBox("docker", NewBox)
}
