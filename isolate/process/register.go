package process

import "github.com/interiorem/stout/isolate"

func init() {
	isolate.RegisterBox("process", NewBox)
}
