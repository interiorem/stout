package process

import "github.com/noxiouz/stout/isolate"

func init() {
	isolate.RegisterBox("process", NewBox)
}
