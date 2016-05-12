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
	service *cocaine.Service
	locator []string
}

func (st *cocaineCodeStorage) lazyStorageCreate(ctx context.Context) (err error) {
	defer apexctx.GetLogger(ctx).Trace("connect to 'storage' service").Stop(&err)

	st.m.Lock()
	defer st.m.Unlock()
	if st.service != nil {
		return nil
	}

	var service *cocaine.Service
	service, err = cocaine.NewService(ctx, "storage", st.locator)
	if err != nil {
		return err
	}

	st.service = service

	return nil
}

func (st *cocaineCodeStorage) Spool(ctx context.Context, appname string) (data []byte, err error) {
	if err = st.lazyStorageCreate(ctx); err != nil {
		return nil, err
	}
	defer apexctx.GetLogger(ctx).WithField("app", appname).Trace("read code from storage").Stop(&err)

	channel, err := st.service.Call(ctx, "read", "apps", appname)
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
