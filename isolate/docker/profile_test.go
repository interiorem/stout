package docker

import (
	"testing"

	"github.com/noxiouz/stout/isolate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileDecodeTo(t *testing.T) {
	rrequire := require.New(t)

	// {
	//     "type": "docker",
	//     "registry": "someregistry",
	//     "repository": "somerepository",
	// 	"endpoint": "someendpoint",
	//
	//     "network_mode": "somenetwork",
	// 	"runtime-path": "someruntimepath",
	//     "cwd": "somecwd",
	//
	// 	"tmpfs": {"tmpfsa": "sometmpfsoption"},
	//     "binds": ["bindA", "bindB"],
	//
	// 	"resources": {
	// 		"memory": 100,
	// 		"CpuShares": 1024
	// 	}
	// }
	var msgpacked = []byte{138, 172, 110, 101, 116, 119, 111, 114, 107, 95, 109, 111, 100, 101, 171, 115, 111, 109,
		101, 110, 101, 116, 119, 111, 114, 107, 165, 98, 105, 110, 100, 115, 146, 165, 98, 105, 110,
		100, 65, 165, 98, 105, 110, 100, 66, 168, 101, 110, 100, 112, 111, 105, 110, 116, 172, 115, 111,
		109, 101, 101, 110, 100, 112, 111, 105, 110, 116, 168, 114, 101, 103, 105, 115, 116, 114, 121, 172,
		115, 111, 109, 101, 114, 101, 103, 105, 115, 116, 114, 121, 170, 114, 101, 112, 111, 115, 105, 116,
		111, 114, 121, 174, 115, 111, 109, 101, 114, 101, 112, 111, 115, 105, 116, 111, 114, 121, 172, 114,
		117, 110, 116, 105, 109, 101, 45, 112, 97, 116, 104, 175, 115, 111, 109, 101, 114, 117, 110, 116,
		105, 109, 101, 112, 97, 116, 104, 165, 116, 109, 112, 102, 115, 129, 166, 116, 109, 112, 102, 115,
		97, 175, 115, 111, 109, 101, 116, 109, 112, 102, 115, 111, 112, 116, 105, 111, 110, 164, 116, 121,
		112, 101, 166, 100, 111, 99, 107, 101, 114, 163, 99, 119, 100, 167, 115, 111, 109, 101, 99, 119, 100,
		169, 114, 101, 115, 111, 117, 114, 99, 101, 115, 130, 169, 67, 112, 117, 83, 104, 97, 114, 101, 115, 205,
		4, 0, 166, 109, 101, 109, 111, 114, 121, 100}

	opts := isolate.NewRawProfileFromBytes(msgpacked)

	var profile = new(Profile)
	rrequire.NoError(opts.DecodeTo(profile))

	asrt := assert.New(t)
	asrt.Equal(profile.Registry, "someregistry")
	asrt.Equal(profile.Repository, "somerepository")
	asrt.Equal(profile.Endpoint, "someendpoint")
	asrt.Equal(profile.NetworkMode, "somenetwork")
	asrt.Equal(profile.RuntimePath, "someruntimepath")
	asrt.Equal(profile.Cwd, "somecwd")
	asrt.Equal(profile.Binds, []string{"bindA", "bindB"})
	asrt.Equal(profile.Tmpfs, map[string]string{"tmpfsa": "sometmpfsoption"})

	res := profile.Resources

	memlimit, _ := res.Memory.Int()
	asrt.Equal(memlimit, int64(100))
	cpuShares, _ := res.CPUShares.Int()
	asrt.Equal(cpuShares, int64(1024))
}
