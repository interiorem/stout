package porto

import (
	"encoding/json"
	"testing"

	"github.com/interiorem/stout/isolate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Registry   string `msg:"registry"`
// Repository string `msg:"repository"`
//
// NetworkMode string `msg:"network_mode"`
// Cwd         string `msg:"cwd"`
//
// Binds []string `msg:"binds"`
//
// Container    map[string]string `msg:"container"`
// Volume       map[string]string `msg:"volume"`
// ExtraVolumes []VolumeProfile   `msg:"extravolumes"`

func TestProfileDecodable(t *testing.T) {
	rrequire := require.New(t)
	const jsonprofile = `
    {
        "type": "porto",
        "registry": "someregistry",
        "repository": "somerepository",

        "network_mode": "somenetwork",
        "cwd": "somecwd",
        "binds": ["bindA", "bindB"],

        "container": {
            "env": "someenv"
        },
        "volume": {
            "volumeopt": "somevolumeopt"
        },
        "extravolumes": [{"target": "sometarget", "properties": {"storage": "somestorage", "backend": "bind"}}]
    }
    `

	var mp map[string]interface{}
	rrequire.NoError(json.Unmarshal([]byte(jsonprofile), &mp))
	rrequire.True(len(mp) > 0)

	opts, err := isolate.NewRawProfile(mp)
	rrequire.NoError(err)

	var profile = new(Profile)
	rrequire.NoError(opts.DecodeTo(profile))

	asrt := assert.New(t)
	asrt.Equal(profile.Registry, "someregistry")
	asrt.Equal(profile.Repository, "somerepository")
	asrt.Equal(profile.NetworkMode, "somenetwork")
	asrt.Equal(profile.Cwd, "somecwd")
	asrt.Equal(profile.Binds, []string{"bindA", "bindB"})
	asrt.Equal(profile.Container, map[string]string{"env": "someenv"})
	asrt.Equal(profile.Volume, map[string]string{"volumeopt": "somevolumeopt"})

	exv := profile.ExtraVolumes
	rrequire.True(len(exv) == 1)
	asrt.Equal(exv[0].Target, "sometarget")
	asrt.Equal(exv[0].Properties, map[string]string{"storage": "somestorage", "backend": "bind"})
}
