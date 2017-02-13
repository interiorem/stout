package process

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"testing"

	"github.com/noxiouz/stout/isolate"
	"github.com/noxiouz/stout/isolate/testsuite"

	"github.com/noxiouz/stout/pkg/log"
	"golang.org/x/net/context"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

func init() {
	f := func(c *C) isolate.RawProfile {
		opts, err := isolate.NewRawProfile(&Profile{})
		if err != nil {
			c.Fatalf("can not create raw profile %v", err)
		}
		return opts
	}
	testsuite.RegisterSuite(processBoxConstructorWithMockedStorage, f, testsuite.NeverSkip)
}

func TestProfileDecodeTo(t *testing.T) {
	mp := map[string]interface{}{
		"spool": "somespool",
	}
	opts, err := isolate.NewRawProfile(mp)
	if err != nil {
		t.Fatalf("unable to encode test profile as JSON %v", err)
	}

	var profile = new(Profile)
	if err = opts.DecodeTo(profile); err != nil {
		t.Fatalf("DecodeTo failed %v", err)
	}

	if profile.Spool != mp["spool"] {
		t.Fatalf("Spool is expected to be %s, not %s ", mp["spool"], profile.Spool)
	}
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
	old := createCodeStorage
	createCodeStorage = func(locator []string) codeStorage {
		return &mockCodeStorage{
			files: map[string][]byte{
				"worker": makeGzipedArch(c),
			},
		}
	}
	defer func() { createCodeStorage = old }()

	return NewBox(context.Background(), isolate.BoxConfig{"spool": c.MkDir()})
}
