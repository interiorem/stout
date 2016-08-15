package porto

import (
	"bytes"
	"net"
	"sync"
	"sync/atomic"
	"time"

	porto "github.com/yandex/porto/src/api/go"
	portorpc "github.com/yandex/porto/src/api/go/rpc"
)

var (
	propLock            sync.Mutex
	containerProperties atomic.Value
	containerData       atomic.Value
)

func init() {
	containerData.Store([]string{})
	containerProperties.Store([]string{})
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

type containerFootprint struct {
	portoConn   porto.API
	containerID string
}

func (c containerFootprint) String() string {
	buff := newBuff()
	defer buffPool.Put(buff)

	properties := containerProperties.Load().([]string)
	if len(properties) == 0 {
		func() {
			propLock.Lock()
			defer propLock.Unlock()
			if len(containerProperties.Load().([]string)) != 0 {
				return
			}
			portoProps, err := c.portoConn.Plist()
			if err != nil {
				return
			}
			properties = make([]string, 0, len(portoProps))
			for _, property := range portoProps {
				properties = append(properties, property.Name)
			}
			containerProperties.Store(properties)
		}()
	}

	data := containerData.Load().([]string)
	if len(data) == 0 {
		func() {
			propLock.Lock()
			defer propLock.Unlock()
			if len(containerData.Load().([]string)) != 0 {
				return
			}
			portoData, err := c.portoConn.Dlist()
			if err != nil {
				return
			}
			data = make([]string, 0, len(portoData))
			for _, dataItem := range portoData {
				data = append(data, dataItem.Name)
			}
			containerData.Store(data)
		}()
	}

	for _, property := range properties {
		value, err := c.portoConn.GetProperty(c.containerID, property)
		buff.WriteByte(' ')
		buff.WriteString(property)
		buff.WriteByte('=')
		if err != nil {
			buff.WriteString(err.Error())
		} else {
			buff.WriteString(value)
		}
		buff.WriteByte('\n')
	}

	for _, dt := range data {
		if dt == "stderr" || dt == "stdout" {
			continue
		}

		value, err := c.portoConn.GetData(c.containerID, dt)
		buff.WriteByte(' ')
		buff.WriteString(dt)
		buff.WriteByte('=')
		if err != nil {
			buff.WriteString(err.Error())
		} else {
			buff.WriteString(value)
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
