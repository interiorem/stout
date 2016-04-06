package process

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"

	"github.com/noxiouz/stout/isolate"

	"golang.org/x/net/context"
)

func unpackArchive(ctx context.Context, data []byte, target string) (err error) {
	log := isolate.GetLogger(ctx).WithField("target", target)
	defer log.Trace("unpacking an archive").Stop(&err)

	if err = os.RemoveAll(target); err != nil {
		return err
	}

	if err = os.Mkdir(target, 0755); err != nil {
		log.Error("unable to create spool directory")
		return err
	}

	gzR, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer gzR.Close()

	tr := tar.NewReader(gzR)
	for {
		hdr, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		path := filepath.Join(target, hdr.Name)
		info := hdr.FileInfo()
		if info.IsDir() {
			log.Debugf("unpackArchive: unpack directory %s (size %d) to %s", hdr.Name, hdr.Size, path)
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}

		log.Debugf("unpackArchive: unpack %s (size %d) to %s", hdr.Name, hdr.Size, path)
		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		nn, err := io.Copy(file, tr)
		log.Debugf("unpackArchive: extracted (%d/%d) bytes of %s: %v", nn, hdr.Size, path, err)
		file.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
