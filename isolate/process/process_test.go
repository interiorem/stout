package process

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"testing"

	"github.com/noxiouz/stout/isolate"
	"github.com/noxiouz/stout/isolate/testsuite"

	"golang.org/x/net/context"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

func init() {
	opts := make(isolate.Profile)
	testsuite.RegisterSuite(processBoxConstructorWithMockedStorage, opts, testsuite.NeverSkip)
}

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

func makeGzipedArch(c *C) []byte {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	files := []struct {
		Name, Body string
		Mode       int64
	}{
		{"worker.sh", testsuite.ScriptWorkerSh, 0777},
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

func processBoxConstructorWithMockedStorage(c *C) (isolate.Box, error) {
	box := &Box{
		spoolPath: c.MkDir(),
		storage: &mockCodeStorage{
			files: map[string][]byte{
				"worker": makeGzipedArch(c),
			},
		},
	}

	return box, nil
}
