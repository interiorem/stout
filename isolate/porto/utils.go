package porto

import (
	"bytes"
	"sync"

	porto "github.com/yandex/porto/src/api/go"
	portorpc "github.com/yandex/porto/src/api/go/rpc"
)

func isEqualPortoError(err error, expectedErrno portorpc.EError) bool {
	switch err := err.(type) {
	case (*porto.Error):
		return err.Errno == expectedErrno
	default:
		return false
	}
}

var (
	buffPool = sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}
)

func newBuff() *bytes.Buffer {
	buff := buffPool.Get().(*bytes.Buffer)
	buff.Reset()
	return buff
}
