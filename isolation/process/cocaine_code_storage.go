package process

import (
	"sync"
	"sync/atomic"

	"golang.org/x/net/context"

	cocaine "github.com/cocaine/cocaine-framework-go/cocaine12"

	"github.com/noxiouz/stout/isolation"
)

type cocaineCodeStorage struct {
	m           sync.Mutex
	onceCreated uint32
	service     *cocaine.Service
}

func (st *cocaineCodeStorage) lazyStorageCreate(ctx context.Context) (err error) {
	if atomic.LoadUint32(&st.onceCreated) == 1 {
		return nil
	}
	defer isolation.GetLogger(ctx).Trace("connect to 'storage' service").Stop(&err)

	st.m.Lock()
	defer st.m.Unlock()
	if st.onceCreated == 0 {
		defer atomic.StoreUint32(&st.onceCreated, 1)

		var service *cocaine.Service
		service, err = cocaine.NewService(ctx, "storage", nil)
		if err != nil {
			return err
		}

		st.service = service
	}

	return nil
}

func (st *cocaineCodeStorage) Spool(ctx context.Context, appname string) (data []byte, err error) {
	if err = st.lazyStorageCreate(ctx); err != nil {
		return nil, err
	}
	defer isolation.GetLogger(ctx).Trace("read code from storage").WithField("app", appname).Stop(&err)

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
