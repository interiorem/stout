package docker

import "github.com/interiorem/stout/isolate"

func init() {
	isolate.RegisterBox("docker", NewBox)
}
