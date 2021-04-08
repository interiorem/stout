package porto

import "github.com/interiorem/stout/isolate"

func init() {
	isolate.RegisterBox("porto", NewBox)
}
