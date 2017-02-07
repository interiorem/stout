package porto

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"testing"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/noxiouz/stout/isolate"
	"github.com/noxiouz/stout/isolate/testsuite"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

func init() {
	f := func(c *C) isolate.RawProfile {
		opts := map[string]string{
			"registry": "http://localhost:5000",
			"cwd":      "/usr/bin",
		}

		r, err := isolate.NewRawProfile(opts)
		if err != nil {
			c.Fatalf("unable to create raw profile %v", err)
		}
		return r
	}

	testsuite.RegisterSuite(portoBoxConstructor, f, func() string {
		if os.Getenv("TRAVIS") == "true" {
			return "Skip Porto tests under Travis CI"
		}
		if runtime.GOOS != "linux" {
			return "Porto tests could be run in Linux only"
		}
		return ""
	})
}

func buildTestImage(c *C) {
	var endpoint string
	if endpoint = os.Getenv("DOCKER_HOST"); endpoint == "" {
		endpoint = client.DefaultDockerHost
	}

	const dockerFile = `
FROM alpine

COPY worker.sh /usr/bin/worker.sh
	`
	cl, err := client.NewClient(endpoint, "", nil, nil)
	c.Assert(err, IsNil)

	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	files := []struct {
		Name, Body string
		Mode       int64
	}{
		{"worker.sh", testsuite.ScriptWorkerSh, 0777},
		{"Dockerfile", dockerFile, 0666},
	}

	for _, file := range files {
		hdr := &tar.Header{
			Name: file.Name,
			Mode: file.Mode,
			Size: int64(len(file.Body)),
		}
		c.Assert(tw.WriteHeader(hdr), IsNil)
		_, err = tw.Write([]byte(file.Body))
		c.Assert(err, IsNil)
	}
	c.Assert(tw.Close(), IsNil)

	opts := types.ImageBuildOptions{
		Tags: []string{"worker"},
	}

	resp, err := cl.ImageBuild(context.Background(), buf, opts)
	c.Assert(err, IsNil)
	defer resp.Body.Close()
	io.Copy(ioutil.Discard, resp.Body)

	err = cl.ImageTag(context.Background(), "worker", "localhost:5000/worker", types.ImageTagOptions{Force: true})
	c.Assert(err, IsNil)
	buildResp, err := cl.ImagePush(context.Background(), "localhost:5000/worker:latest", types.ImagePushOptions{RegistryAuth: "e30="})
	c.Assert(err, IsNil)
	defer buildResp.Close()
	io.Copy(ioutil.Discard, buildResp)
}

func portoBoxConstructor(c *C) (isolate.Box, error) {
	buildTestImage(c)
	cfg := isolate.BoxConfig{
		"layers":     "/tmp/layers",
		"containers": "/tmp/containers",
		"journal":    "/tmp/portojournal.jrnl",
	}

	b, err := NewBox(context.Background(), cfg)
	c.Assert(err, IsNil)
	return b, err
}
