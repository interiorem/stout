package process

import (
	"sync"

	"golang.org/x/net/context"

	cocaine "github.com/cocaine/cocaine-framework-go/cocaine12"
	"github.com/ugorji/go/codec"

	apexctx "github.com/m0sth8/context"
)

type cocaineCodeStorage struct {
	m       sync.Mutex
	locator []string
}

func (st *cocaineCodeStorage) createStorage(ctx context.Context) (service *cocaine.Service, err error) {
	defer apexctx.GetLogger(ctx).Trace("connect to 'storage' service").Stop(&err)
	return cocaine.NewService(ctx, "storage", st.locator)
}

func (st *cocaineCodeStorage) Spool(ctx context.Context, appname string) (data []byte, err error) {
	storage, err := st.createStorage(ctx)
	if err != nil {
		return nil, err
	}
	defer storage.Close()
	defer apexctx.GetLogger(ctx).WithField("app", appname).Trace("read code from storage").Stop(&err)

	channel, err := storage.Call(ctx, "read", "apps", appname)
	if err != nil {
		return nil, err
	}

	res, err := channel.Get(ctx)
	if err != nil {
		return nil, err
	}

	var raw []byte
	if err = res.ExtractTuple(&raw); err != nil {
		return nil, err
	}

	if err = codec.NewDecoderBytes(raw, &codec.MsgpackHandle{}).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}
