package porto

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmizerany/assert"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/noxiouz/stout/isolate/docker"
	"github.com/stretchr/testify/require"
	portorpc "github.com/yandex/porto/src/api/go/rpc"
	"golang.org/x/net/context"
)

func TestContainer(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()

	var endpoint string
	if endpoint = os.Getenv("DOCKER_HOST"); endpoint == "" {
		endpoint = client.DefaultDockerHost
	}
	client, err := client.NewClient(endpoint, "", nil, nil)
	require.NoError(err)

	imgs, err := client.ImageList(ctx, types.ImageListOptions{MatchName: "alpine"})
	require.NoError(err)
	if len(imgs) == 0 {
		resp, err := client.ImagePull(ctx, "alpine", types.ImagePullOptions{})
		require.NoError(err)
		io.Copy(ioutil.Discard, resp)
		resp.Close()
	}

	dir, err := ioutil.TempDir("", "")
	require.NoError(err)
	defer os.RemoveAll(dir)

	resp, err := client.ImageSave(ctx, []string{"alpine"})
	require.NoError(err)
	defer resp.Close()

	imagetar := filepath.Join(dir, "alpine.tar.gz")
	fi, err := os.Create(imagetar)
	require.NoError(err)
	defer fi.Close()
	_, err = io.Copy(fi, resp)
	require.NoError(err)
	fi.Close()
	resp.Close()

	var profile = docker.Profile{
		RuntimePath: "/var/run",
		NetworkMode: "host",
		Cwd:         "/tmp",
		Resources: docker.Resources{
			Memory: 4 * 1024 * 1024,
		},
		Tmpfs: map[string]string{
			"/tmp/a": "size=100000",
		},
	}

	portoConn, err := portoConnect()
	if err != nil {
		t.Fatal(err)
	}
	defer portoConn.Close()

	err = portoConn.ImportLayer("testalpine", imagetar, false)
	if err != nil {
		require.True(isEqualPortoError(err, portorpc.EError_LayerAlreadyExists))
	}

	ei := execInfo{
		Profile:    &profile,
		name:       "TestContainer",
		executable: "echo",
		args:       map[string]string{"--endpoint": "/var/run/cocaine.sock"},
		env:        map[string]string{"A": "B"},
	}

	cfg := containerConfig{
		Root:  dir,
		ID:    "LinuxAlpine",
		Layer: "testalpine",
	}

	cnt, err := newContainer(ctx, portoConn, cfg, ei)
	require.NoError(err)
	require.NoError(cnt.start(portoConn, ioutil.Discard))
	defer cnt.Kill()

	env, err := portoConn.GetProperty(cnt.containerID, "env")
	require.NoError(err)
	assert.Equal(t, "A:B", env)

	command, err := portoConn.GetProperty(cnt.containerID, "command")
	require.NoError(err)
	assert.Equal(t, "echo --endpoint /var/run/cocaine.sock", command)

	cwd, err := portoConn.GetProperty(cnt.containerID, "cwd")
	require.NoError(err)
	assert.Equal(t, profile.Cwd, cwd)
}
