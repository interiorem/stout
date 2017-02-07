package porto

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	portorpc "github.com/yandex/porto/src/api/go/rpc"
)

func TestExecInfoFormatters(t *testing.T) {
	assert := assert.New(t)
	info := execInfo{
		name:       "testapp",
		executable: "/usr/bin/testapp",
		args: map[string]string{
			"--endpoint": "/var/run/cocaine.sock",
			"--app":      "testapp",
			"protocol":   "1",
			"--locator":  "[2a02:6b8:0:1605::32]:10053,5.45.197.172:10053",
			"--uuid":     "bfe13176-7195-47db-a469-1b73b25d18c2",
		},
		env: map[string]string{
			"envA": "A",
			"envB": "B",
		},
		// Profile: &docker.Profile{
		// 	RuntimePath: "/var/run",
		// 	NetworkMode: "host",
		// 	Cwd:         "/tmp",
		// 	Resources: docker.Resources{
		// 		Memory: 4 * 1024 * 1024,
		// 	},
		// 	Tmpfs: map[string]string{
		// 		"/tmp/a": "size=100000",
		// 	},
		// 	Binds: []string{"/tmp:/bind:rw"},
		// },
		Profile: &Profile{
			Binds:       []string{"/tmp:/bind:rw"},
			Cwd:         "/tmp",
			NetworkMode: "host",
		},
	}

	assert.Equal("/var/run/cocaine.sock /run/cocaine;/tmp /bind rw", formatBinds(&info))
	env := strings.Split(formatEnv(info.env), ";")
	sort.Strings(env)
	assert.Equal([]string{"envA=A", "envB=B"}, env)
	cmd := strings.Split(formatCommand(info.executable, info.args), " ")
	assert.Len(cmd, 1+len(info.args)*2)
	assert.Equal(info.executable, cmd[0])

	var found bool
	for i, s := range cmd {
		if s == "--endpoint" {
			found = true
			assert.Equal("/run/cocaine", cmd[i+1])
		}
	}
	assert.True(found)
}

func TestContainer(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skipf("Skip under %s", runtime.GOOS)
		return
	}
	if os.Getenv("TRAVIS") == "true" {
		t.Skip("Skip Porto tests under Travis CI")
		return
	}
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

	imagetar := filepath.Join(dir, "alpine.tar.gz")

	resp, err := client.ImageSave(ctx, []string{"alpine"})
	require.NoError(err)
	defer resp.Close()

	tarReader := tar.NewReader(resp)
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		require.NoError(err)
		if strings.HasSuffix(hdr.Name, "layer.tar") {
			fi, err := os.Create(imagetar)
			require.NoError(err)
			defer fi.Close()
			gz := gzip.NewWriter(fi)
			_, err = io.Copy(gz, tarReader)
			gz.Close()
			require.NoError(err)
			fi.Close()
		}
		io.Copy(ioutil.Discard, tarReader)
	}

	portoConn, err := portoConnect()
	if err != nil {
		t.Fatal(err)
	}
	defer portoConn.Close()

	portoConn.Destroy("IsolateLinuxApline")
	err = portoConn.ImportLayer("testalpine", imagetar, false)
	if err != nil {
		require.True(isEqualPortoError(err, portorpc.EError_LayerAlreadyExists))
	}

	ei := execInfo{
		Profile: &Profile{
			Cwd: "/tmp",
			ExtraVolumes: []VolumeProfile{
				{
					Target: "/tmpfs",
					Properties: map[string]string{
						"backend":     "tmpfs",
						"space_limit": "10000",
					},
				}, {
					Target: "/bind",
					Properties: map[string]string{
						"backend": "bind",
						"storage": dir,
					},
				},
			},
		},
		name:       "TestContainer",
		executable: "echo",
		args:       map[string]string{"--endpoint": "/var/run/cocaine.sock"},
		env:        map[string]string{"A": "B"},
	}

	cfg := containerConfig{
		Root:     dir,
		ID:       "IsolateLinuxApline",
		Layer:    "testalpine",
		execInfo: ei,
	}

	cnt, err := newContainer(ctx, portoConn, cfg)
	cnt.cleanupEnabled = true
	require.NoError(err)
	require.NoError(cnt.start(portoConn, ioutil.Discard))

	if cnt.cleanupEnabled {
		defer func() {
			for _, vol := range ei.Profile.ExtraVolumes {
				if storage := vol.Properties["storage"]; storage != "" {
					_, err = os.Stat(storage)
					require.True(os.IsNotExist(err), "extra volume %s was not cleaned up", vol.Target)
				}
			}

			_, err = os.Stat(filepath.Join(cfg.Root, "volume"))
			require.True(os.IsNotExist(err), "root volume was not cleaned up")
		}()
	}
	defer cnt.Kill()

	for _, vol := range ei.Profile.ExtraVolumes {
		// Test that tmpfs has been created
		expectedVolPath := filepath.Join(dir, "volume", vol.Target)
		targetDirStat, serr := os.Stat(expectedVolPath)
		require.NoError(serr)
		require.True(targetDirStat.IsDir())
		if storage := vol.Properties["storage"]; storage != "" {
			// containerid is appended to storage
			require.True(strings.HasSuffix(storage, cnt.containerID))
			storageDirStat, zerr := os.Stat(storage)
			require.NoError(zerr)
			require.True(storageDirStat.IsDir())
		}
	}

	env, err := portoConn.GetProperty(cnt.containerID, "env")
	require.NoError(err)
	assert.Equal(t, "A=B", env)

	command, err := portoConn.GetProperty(cnt.containerID, "command")
	require.NoError(err)
	// NOTE: porto can bind a single file inside container, not only a directory
	assert.Equal(t, "echo --endpoint /run/cocaine", command)

	cwd, err := portoConn.GetProperty(cnt.containerID, "cwd")
	require.NoError(err)
	assert.Equal(t, ei.Profile.Cwd, cwd)
}
