package porto

import (
	"fmt"

	"github.com/noxiouz/stout/isolate"

	apexctx "github.com/m0sth8/context"
	"golang.org/x/net/context"

	porto "github.com/yandex/porto/src/api/go"
)

type portoProfile struct {
	isolate.Profile
}

func (p portoProfile) Binds() []string {
	binds, ok := p.Profile["binds"]
	if !ok {
		return nil
	}

	switch binds := binds.(type) {
	case []string:
		return binds
	case [][]byte:
		bs := make([]string, 0, len(binds))
		for _, b := range binds {
			bs = append(bs, string(b))
		}
		return bs
	default:
		return nil
	}
}

func (p portoProfile) Registry() string {
	return getStringValue(p.Profile, "registry")
}

func (p portoProfile) Cwd() string {
	return getStringValue(p.Profile, "cwd")
}

func (p portoProfile) NetworkMode() string {
	return getStringValue(p.Profile, "network_mode")
}

func getStringValue(mp map[string]interface{}, key string) string {
	value, ok := mp[key]
	if !ok {
		return ""
	}

	switch value := value.(type) {
	case string:
		return value
	case []byte:
		return string(value)
	default:
		return ""
	}
}

func (p portoProfile) applyContainerLimits(ctx context.Context, portoConn porto.API, id string) error {
	limits, ok := p.Profile["container"]
	if !ok {
		apexctx.GetLogger(ctx).WithField("container", id).Info("no container limits")
		return nil
	}

	switch limits := limits.(type) {
	case map[string]interface{}:
		log := apexctx.GetLogger(ctx).WithField("container", id)
		for limit, value := range limits {
			strvalue := fmt.Sprintf("%s", value)
			log.Debugf("apply %s %s", limit, strvalue)
			if err := portoConn.SetProperty(id, limit, strvalue); err != nil {
				return err
			}
		}

		return nil
	default:
		return fmt.Errorf("invalid resources type %T", limits)
	}
}

func (p portoProfile) applyVolumeLimits(ctx context.Context, id string, vp map[string]string) error {
	limits, ok := p.Profile["volume"]
	if !ok {
		apexctx.GetLogger(ctx).WithField("container", id).Info("no volume limits")
		return nil
	}

	switch limits := limits.(type) {
	case map[string]interface{}:
		log := apexctx.GetLogger(ctx).WithField("container", id)
		for limit, value := range limits {
			strvalue := fmt.Sprintf("%s", value)
			log.Debugf("apply volume limit %s %s", limit, strvalue)
			vp[limit] = strvalue
		}

		return nil
	default:
		return fmt.Errorf("invalid resources type %T", limits)
	}

	return nil
}
