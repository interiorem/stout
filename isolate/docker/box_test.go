package docker

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/noxiouz/stout/isolate"
	"github.com/noxiouz/stout/isolate/testsuite"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"golang.org/x/net/context"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

func init() {
	var endpoint string
	if endpoint = os.Getenv("DOCKER_HOST"); endpoint == "" {
		endpoint = client.DefaultDockerHost
	}
	opts := isolate.Profile{
		"endpoint": endpoint,
		"cwd":      "/usr/bin",
	}

	testsuite.RegisterSuite(dockerBoxConstructor, opts, testsuite.NeverSkip)
}

func buildTestImage(c *C, endpoint string) {
	const dockerFile = `
FROM ubuntu:trusty

COPY worker.sh /usr/bin/worker.sh
	`
	cl, err := client.NewClient(endpoint, dockerAPIVersion, nil, nil)
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
		Tags:    []string{"worker"},
		Context: buf,
	}
	resp, err := cl.ImageBuild(context.Background(), opts)
	c.Assert(err, IsNil)
	defer resp.Body.Close()

	for {
		var p = make([]byte, 1024)
		_, err := resp.Body.Read(p)
		if err != nil {
			c.Assert(err, Equals, io.EOF)
			break
		}
	}
}

func dockerBoxConstructor(c *C) (isolate.Box, error) {
	var endpoint string
	if endpoint = os.Getenv("DOCKER_HOST"); endpoint == "" {
		endpoint = client.DefaultDockerHost
	}

	buildTestImage(c, endpoint)
	b, err := NewBox(isolate.BoxConfig{
		"endpoint": endpoint,
	})
	c.Assert(err, IsNil)
	return b, err
}
