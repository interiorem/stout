package process

import (
	"sync"

	"golang.org/x/net/context"

	cocaine "github.com/cocaine/cocaine-framework-go/cocaine12"

	"github.com/noxiouz/stout/isolate"
)

type cocaineCodeStorage struct {
	m       sync.Mutex
	service *cocaine.Service
}

func (st *cocaineCodeStorage) lazyStorageCreate(ctx context.Context) (err error) {
	defer isolate.GetLogger(ctx).Trace("connect to 'storage' service").Stop(&err)

	st.m.Lock()
	defer st.m.Unlock()

	var service *cocaine.Service
	service, err = cocaine.NewService(ctx, "storage", nil)
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
	defer isolate.GetLogger(ctx).WithField("app", appname).Trace("read code from storage").Stop(&err)

	channel, err := st.service.Call(ctx, "read", "apps", appname)
	if err != nil {
		return nil, err
	}

	res, err := channel.Get(ctx)
	if err != nil {
		return nil, err
	}

	return data, res.ExtractTuple(&data)
}
