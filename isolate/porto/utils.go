package porto

import (
	"bytes"
	"encoding/json"
	"net"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/docker/distribution"
	porto "github.com/yandex/porto/src/api/go"
	portorpc "github.com/yandex/porto/src/api/go/rpc"
)

var (
	propLock                   sync.Mutex
	containerPropertiesAndData atomic.Value
)

func init() {
	containerPropertiesAndData.Store([]string{})
}

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

type portoData map[string]porto.TPortoGetResponse

var _ json.Marshaler = portoData{}

func (d portoData) MarshalJSON() ([]byte, error) {
	buff := new(bytes.Buffer)
	if d == nil {
		buff.WriteString("null")
		return buff.Bytes(), nil
	}

	keys := containerPropertiesAndData.Load().([]string)
	if len(keys) == 0 {
		keys = make([]string, 0, len(d))
		for k := range d {
			keys = append(keys, k)
		}
		sort.Strings(keys)
	}

	buff.WriteByte('{')
	for i, name := range keys {
		// NOTE: it's unlikely that name is absent in the map
		// but zero value is valid empty string with 0 error code
		v := d[name]
		if i > 0 {
			buff.WriteByte(',')
		}

		buff.WriteByte('"')
		buff.WriteString(name)
		buff.WriteByte('"')
		buff.WriteByte(':')
		buff.WriteByte('"')
		if v.Error == 0 {
			buff.WriteString(v.Value)
		} else {
			buff.WriteString(v.ErrorMsg)
		}
		buff.WriteByte('"')
	}
	buff.WriteByte('}')
	return buff.Bytes(), nil
}

func getPListAndDlist(portoConn porto.API) (list []string) {
	list = containerPropertiesAndData.Load().([]string)
	if len(list) == 0 {
		func() {
			propLock.Lock()
			defer propLock.Unlock()

			portoProps, err := portoConn.Plist()
			if err != nil {
				return
			}
			for _, property := range portoProps {
				list = append(list, property.Name)
			}

			portoData, err := portoConn.Dlist()
			if err != nil {
				return
			}
			for _, dataItem := range portoData {
				if name := dataItem.Name; name != "stdout" && name != "stderr" {
					list = append(list, name)
				}
			}
			sort.Strings(list)
			containerPropertiesAndData.Store(list)
		}()
	}

	return list
}

type containerFootprint struct {
	portoConn   porto.API
	containerID string
}

func (c containerFootprint) String() string {
	buff := newBuff()
	defer buffPool.Put(buff)

	list := getPListAndDlist(c.portoConn)
	data, err := c.portoConn.Get([]string{c.containerID}, list)
	if err != nil {
		return err.Error()
	}

	for name, value := range data[c.containerID] {
		buff.WriteByte(' ')
		buff.WriteString(name)
		buff.WriteByte('=')
		if err != nil {
			buff.WriteString(err.Error())
		} else {
			buff.WriteString(value.Value)
		}
		buff.WriteByte('\n')
	}

	return buff.String()
}

// NOTE: it's dummy connet_with_retyr implementation
// It's subject to replace
func portoConnect() (porto.API, error) {
	for {
		conn, err := porto.Connect()
		if err == nil {
			return conn, nil
		}

		if ne, ok := err.(net.Error); ok && ne.Temporary() {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		return nil, err
	}
}

type layersOrder func(references []distribution.Descriptor) []distribution.Descriptor

var layerOrderV2 layersOrder = func(references []distribution.Descriptor) []distribution.Descriptor {
	return func(slice []distribution.Descriptor) []distribution.Descriptor {
		size := len(slice) - 1
		for i := 0; i < len(slice)/2; i++ {
			slice[i], slice[size-i] = slice[size-i], slice[i]
		}
		return slice
	}(references)
}

var layerOrderV1 layersOrder = func(references []distribution.Descriptor) []distribution.Descriptor {
	return references
}
