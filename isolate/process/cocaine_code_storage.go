package process

import (
	"context"
	"fmt"
	"sync"
	"time"

	cocaine "github.com/cocaine/cocaine-framework-go/cocaine12"
	"github.com/noxiouz/stout/pkg/log"
	"github.com/uber-go/zap"

	"github.com/tinylib/msgp/msgp"
)

type cocaineCodeStorage struct {
	m       sync.Mutex
	locator []string
}

func (st *cocaineCodeStorage) createStorage(ctx context.Context) (*cocaine.Service, error) {
	start := time.Now()
	log.G(ctx).Info("connecting to 'storage' service", zap.Object("locator", st.locator))
	storage, err := cocaine.NewService(ctx, "storage", st.locator)
	if err != nil {
		log.G(ctx).Info("failed to connect to 'storage' service", zap.Error(err), zap.Duration("duration", time.Now().Sub(start)), zap.Object("locator", st.locator))
		return nil, err
	}

	log.G(ctx).Info("connected to 'storage' service successfully", zap.Duration("duration", time.Now().Sub(start)))
	return storage, nil
}

func (st *cocaineCodeStorage) Spool(ctx context.Context, appname string) (data []byte, err error) {
	storage, err := st.createStorage(ctx)
	if err != nil {
		return nil, err
	}
	defer storage.Close()
	defer func() {
		if err != nil {
			log.G(ctx).Error("failed to read code from storage", zap.Error(err), zap.String("app", appname))
		}
	}()

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
		log.G(ctx).Warn("some data left unpacked", zap.String("app", appname), zap.Int("size", len(rest)))
	}
	return data, err
}
