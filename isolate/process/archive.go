package process

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/noxiouz/stout/pkg/log"
	"golang.org/x/net/context"
)

type archiveConstructor func(io.Reader) (io.ReadCloser, error)

func gzipReader(r io.Reader) (io.ReadCloser, error) {
	return gzip.NewReader(r)
}

func zlibReader(r io.Reader) (io.ReadCloser, error) {
	return zlib.NewReader(r)
}

func fallbackTarReader(r io.Reader) (io.ReadCloser, error) {
	return ioutil.NopCloser(r), nil
}

var constructors = []archiveConstructor{
	gzipReader,
	zlibReader,
	fallbackTarReader,
}

func unpackArchive(ctx context.Context, data []byte, target string) (err error) {
	logger := log.G(ctx).WithField("target", target)
	defer logger.Trace("unpacking an archive").Stop(&err)

	if err = os.RemoveAll(target); err != nil {
		return err
	}

	if err = os.Mkdir(target, 0755); err != nil {
		logger.Error("unable to create spool directory")
		return err
	}

	var archiveReader io.ReadCloser
	// NOTE: the last element is TarFallback, that always returns nil error
	for _, constructor := range constructors {
		archiveReader, err = constructor(bytes.NewReader(data))
		if err == nil {
			break
		}
	}
	defer archiveReader.Close()

	tr := tar.NewReader(archiveReader)
UNPACK:
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
			logger.Debugf("unpackArchive: unpack directory %s (size %d) to %s", hdr.Name, hdr.Size, path)
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue UNPACK
		}

		// NOTE: some archives don't contain headers with a directory item
		if dirpath := filepath.Dir(path); dirpath != "." {
			_, err = os.Stat(dirpath)
			if err != nil {
				if !os.IsNotExist(err) {
					return err
				}
				if err = os.MkdirAll(dirpath, 0770); err != nil {
					return err
				}
			}
		}

		logger.Debugf("unpackArchive: unpack %s (size %d) to %s", hdr.Name, hdr.Size, path)
		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		nn, err := io.Copy(file, tr)
		logger.Debugf("unpackArchive: extracted (%d/%d) bytes of %s: %v", nn, hdr.Size, path, err)
		file.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
