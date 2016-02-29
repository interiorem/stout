package server

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplySocketPathRewrite(t *testing.T) {
	assert := assert.New(t)
	var (
		dcfg = dockerPluginProfile{
			Cmd: []string{"echo", "--app", "echo",
				"--endpoint", "/run/cocaine/echo.52735",
				"--locator", "77.88.18.222:10053",
				"--protocol", "1",
				"--uuid", "c9668166-fa05-498f-9c4e-ad072aec66b4"},
			Volumes: map[string]json.RawMessage{
				"/run/cocaine/": []byte{125, 123},
			},
		}

		rcfg = rewriteConfig{
			Enabled: true,
			Home:    "/run/cocaine/",
			Target:  "/tmp/cocaine/",
		}
	)

	binds := applySocketPathRewrites(&rcfg, &dcfg)
	assert.Equal("/tmp/cocaine/echo.52735", dcfg.Cmd[4], "command args have not been rewritten")
	assert.Equal(rcfg.Home+" "+rcfg.Target, binds[0])
}
