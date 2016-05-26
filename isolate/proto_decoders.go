package isolate

import "sync"

var (
	msgpackBytePool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 1024)
		},
	}
)
