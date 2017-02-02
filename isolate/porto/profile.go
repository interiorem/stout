package porto

import "github.com/noxiouz/stout/isolate"

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
