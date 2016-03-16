package process

import (
	"io/ioutil"
	"os"
	"path"

	"golang.org/x/net/context"

	"github.com/noxiouz/stout/isolation"
)

const (
	defaultSpoolPath = "/var/spool/cocaine"
	// TODO: remove later
	defaultFileStorage = "/var/lib/cocaine"
)

var (
	_ isolation.Box = &Box{}
)

type Box struct {
	spoolPath   string
	fileStorage string
}

func NewBox(cfg isolation.BoxConfig) (isolation.Box, error) {
	spoolPath, ok := cfg["spool"].(string)
	if !ok {
		spoolPath = defaultSpoolPath
	}

	fileStorage, ok := cfg["fileStorage"].(string)
	if !ok {
		fileStorage = defaultFileStorage
	}

	box := &Box{
		spoolPath:   spoolPath,
		fileStorage: fileStorage,
	}
	return box, nil
}

func (b *Box) Spawn(ctx context.Context, name, executable string, args, env map[string]string) (isolation.Process, error) {
	return nil, nil
}

func (b *Box) Spool(ctx context.Context, name string, opts isolation.Profile) error {
	data, err := b.fetch(ctx, name)
	if err != nil {
		return err
	}

	if isolation.IsCancelled(ctx) {
		return nil
	}

	return unpackArchive(ctx, data, b.spoolPath)
}

func (b *Box) fetch(ctx context.Context, appname string) ([]byte, error) {
	filepath := path.Join(b.fileStorage, "apps", appname)
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ioutil.ReadAll(f)
}
