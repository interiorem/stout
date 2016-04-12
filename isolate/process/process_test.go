package process

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/noxiouz/stout/isolate"

	"golang.org/x/net/context"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

func init() {
	Suite(&processBoxSuite{})
}

const scriptWorkerSh = `#!/usr/bin/env python
import os
import sys

print(' '.join(sys.argv[1:]))
for k, v in os.environ.items():
    print("%s=%s" % (k, v))
`

type mockCodeStorage struct {
	files map[string][]byte
}

func (m *mockCodeStorage) Spool(ctx context.Context, appname string) ([]byte, error) {
	data, ok := m.files[appname]
	if !ok {
		return nil, fmt.Errorf("no such file %s", appname)
	}

	return data, nil
}

type processBoxSuite struct {
	mockStorage *mockCodeStorage
}

func makeGzipedArch(c *C) []byte {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	files := []struct {
		Name, Body string
		Mode       int64
	}{
		{"worker.sh", scriptWorkerSh, 0777},
		{"data.txt", "data.txt", 0666},
	}

	for _, file := range files {
		hdr := &tar.Header{
			Name: file.Name,
			Mode: file.Mode,
			Size: int64(len(file.Body)),
		}
		c.Assert(tw.WriteHeader(hdr), IsNil)
		_, err := tw.Write([]byte(file.Body))
		c.Assert(err, IsNil)
	}
	c.Assert(tw.Close(), IsNil)

	gzipped := new(bytes.Buffer)

	gzwr := gzip.NewWriter(gzipped)
	_, err := gzwr.Write(buf.Bytes())
	c.Assert(err, IsNil)
	gzwr.Close()
	return gzipped.Bytes()
}

func (s *processBoxSuite) SetUpSuite(c *C) {
	s.mockStorage = &mockCodeStorage{
		files: map[string][]byte{
			"worker": makeGzipedArch(c),
		},
	}
}

func (s *processBoxSuite) TestSpawn(c *C) {
	var (
		spoolPath = c.MkDir()
		ctx       = context.Background()

		opts isolate.Profile

		name       = "worker"
		executable = "worker.sh"
		args       = map[string]string{
			"--uuid":     "some_uuid",
			"--locator":  "127.0.0.1:10053",
			"--endpoint": "/var/run/cocaine.sock",
			"--app":      "appname",
		}
		env = map[string]string{
			"enva": "a",
			"envb": "b",
		}
	)

	box := &Box{
		spoolPath: spoolPath,
		storage:   s.mockStorage,
	}

	err := box.Spool(ctx, name, opts)
	c.Assert(err, IsNil)

	pr, err := box.Spawn(ctx, opts, name, executable, args, env)
	c.Assert(err, IsNil)

	first := true
	body := new(bytes.Buffer)
	for inc := range pr.Output() {
		c.Assert(inc.Err, IsNil)
		if first {
			first = false
			c.Assert(inc.Data, HasLen, 0)
		}

		body.Write(inc.Data)
	}

	unsplittedArgs, err := body.ReadString('\n')
	c.Assert(err, IsNil)

	cargs := strings.Split(strings.Trim(unsplittedArgs, "\n"), " ")
	c.Assert(cargs, HasLen, len(args)*2)
	for i := 0; i < len(cargs); {
		c.Assert(args[cargs[i]], Equals, cargs[i+1])
		i += 2
	}

	cenv := make(map[string]string)
	for {
		envline, err := body.ReadString('\n')
		if err == io.EOF {
			break
		}
		envs := strings.Split(envline[:len(envline)-1], "=")
		c.Assert(envs, HasLen, 2)
		cenv[envs[0]] = envs[1]
	}

	for k, v := range env {
		c.Assert(cenv[k], Equals, v)
	}
}
