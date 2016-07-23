package porto

import "github.com/noxiouz/stout/isolate"

func init() {
	isolate.RegisterBox("porto", NewBox)
}
