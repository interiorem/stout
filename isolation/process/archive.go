package process

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/net/context"
)

func unpackArchive(ctx context.Context, data []byte, target string) error {
	buff := bytes.NewReader(data)
	gzR, err := gzip.NewReader(buff)
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
			log.Printf("Unpack directory %s (size %d) to %s", hdr.Name, hdr.Size, path)
			if err := os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}

		log.Printf("Unpack contents of %s (size %d) to %s", hdr.Name, hdr.Size, path)
		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		nn, err := io.Copy(file, tr)
		log.Printf("Extracted (%d/%d) bytes of %s: %v", nn, hdr.Size, path, err)
		file.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
