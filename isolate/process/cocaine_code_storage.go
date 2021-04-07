package process

import (
	"fmt"
	"sync"

	"golang.org/x/net/context"

	cocaine "github.com/cocaine/cocaine-framework-go/cocaine12"

	"github.com/interiorem/stout/pkg/log"

	"github.com/tinylib/msgp/msgp"
)

type cocaineCodeStorage struct {
	m       sync.Mutex
	locator []string
}

func (st *cocaineCodeStorage) createStorage(ctx context.Context) (service *cocaine.Service, err error) {
	defer log.G(ctx).Trace("connect to 'storage' service").Stop(&err)
	return cocaine.NewService(ctx, "storage", st.locator)
}

func (st *cocaineCodeStorage) Spool(ctx context.Context, appname string) (data []byte, err error) {
	storage, err := st.createStorage(ctx)
	if err != nil {
		return nil, err
	}
	defer storage.Close()
	defer log.G(ctx).WithField("app", appname).Trace("read code from storage").Stop(&err)

	channel, err := storage.Call(ctx, "read", "apps", appname)
	if err != nil {
		return nil, err
	}

	res, err := channel.Get(ctx)
	if err != nil {
		return nil, err
	}

	num, val, err := res.Result()
	if err != nil || num != 0 || len(val) != 1 {
		return nil, fmt.Errorf("invalid Storage service reply err: %v, num %d, len(val): %d", err, num, len(val))
	}

	var raw, rest []byte
	raw, ok := val[0].([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid Storage.Read value type %T", val[0])
	}

	switch tp := msgp.NextType(raw); tp {
	case msgp.BinType:
		data, rest, err = msgp.ReadBytesZC(raw)
	case msgp.StrType:
		data, rest, err = msgp.ReadStringZC(raw)
	default:
		return nil, fmt.Errorf("invalid msgpack type for an archive: %s", tp)
	}

	if len(rest) != 0 {
		log.G(ctx).WithField("app", appname).Warnf("Some left unpacked: %d", len(rest))
	}
	return data, err
}
