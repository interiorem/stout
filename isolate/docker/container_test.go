package docker

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/net/context"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"

	"github.com/stretchr/testify/assert"
)

func TestContainer(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()

	var endpoint string
	if endpoint = os.Getenv("DOCKER_HOST"); endpoint == "" {
		endpoint = client.DefaultDockerHost
	}
	client, err := client.NewClient(endpoint, "", nil, defaultHeaders)
	assert.NoError(err)
	version, err := client.ServerVersion(ctx)
	assert.NoError(err)

	imgs, err := client.ImageList(ctx, types.ImageListOptions{MatchName: "ubuntu:trusty"})
	assert.NoError(err)
	if len(imgs) == 0 {
		resp, err := client.ImagePull(ctx, "ubuntu:trusty", types.ImagePullOptions{})
		assert.NoError(err)
		io.Copy(ioutil.Discard, resp)
		resp.Close()
	}

	var profile = Profile{
		RuntimePath: "/var/run",
		NetworkMode: "host",
		Cwd:         "/tmp",
		Resources: Resources{
			Memory: 4 * 1024 * 1024,
		},
		Tmpfs: map[string]string{
			"/tmp/a": "size=100000",
		},
	}

	args := map[string]string{"--endpoint": "/var/run/cocaine.sock"}
	env := map[string]string{"A": "B"}

	container, err := newContainer(ctx, client, &profile, "ubuntu:trusty", "echo", args, env)
	assert.NoError(err)

	inspect, err := client.ContainerInspect(ctx, container.containerID)
	assert.NoError(err)
	assert.Equal([]string{"--endpoint", "/var/run/cocaine.sock"}, inspect.Args, "invalid args")

	ver := strings.SplitN(version.Version, ".", 3)
	v, err := strconv.Atoi(ver[1])
	assert.NoError(err)

	if v >= 10 {
		assert.Equal(profile.Tmpfs, inspect.HostConfig.Tmpfs, "invalid tmpfs value")
	} else {
		t.Logf("%s does not support tmpfs", version.Version)
	}

	assert.Equal(profile.Resources.Memory, inspect.HostConfig.Memory, "invalid memory limit")

	container.Kill()
	_, err = client.ContainerInspect(ctx, container.containerID)
	assert.Error(err)
}

func TestImagePull(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()

	fixtures := []struct {
		name string
		body []byte
		err  error
	}{
		{"NoEOF", []byte(`{"Status": "OK"}`), nil},
		{"LinesCase", []byte("{\"Status\": \"OK\"}\n{\"Status\": \"OK\"}"), nil},
		{"LinesCaseError", []byte("{\"Status\": \"OK\"}\n{\"Status\": \"OK\"}{\"Error\": \"blabla\"}"), fmt.Errorf("blabla")},
		{"FlatCase", []byte("{\"Status\": \"OK\"}{\"Status\": \"OK\"}"), nil},
		{"FlatCaseError", []byte(`{"Status": "OK"}{"Status": "OK"}{"Error": "blabla"}`), fmt.Errorf("blabla")},
		{"MixedCase", []byte("{\"Status\": \"OK\"}\n{\"Status\": \"OK\"}{\"Status\": \"OK\"}"), nil},
		{"MixedCaseError", []byte("{\"Status\": \"OK\"}\n{\"Status\": \"OK\"}{\"Error\": \"blabla\"}"), fmt.Errorf("blabla")},
	}

	for _, fixt := range fixtures {
		err := decodeImagePull(ctx, bytes.NewReader(fixt.body))
		assert.Equal(fixt.err, err, "invalid error for %v", fixt.name)
	}
}
