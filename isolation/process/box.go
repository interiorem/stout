package process

import (
	"log"
	"os"
	"path/filepath"

	"github.com/ugorji/go/codec"
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
	workDir := filepath.Join(b.spoolPath, name)
	execPath := filepath.Join(workDir, executable)
	log.Printf("processBox.Spawn(): name `%s`, executable `%s`, workdir `%s`, exec_path `%s`",
		name, executable, workDir, execPath)

	return newProcess(ctx, execPath, args, env, workDir)
}

func (b *Box) Spool(ctx context.Context, name string, opts isolation.Profile) error {
	log.Printf("processBox.Spool(): name `%s`, profile `%v`", name, opts)
	data, err := b.fetch(ctx, name)
	if err != nil {
		return err
	}

	if isolation.IsCancelled(ctx) {
		return nil
	}

	return unpackArchive(ctx, data, filepath.Join(b.spoolPath, name))
}

func (b *Box) fetch(ctx context.Context, appname string) ([]byte, error) {
	path := filepath.Join(b.fileStorage, "apps", appname)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var data []byte
	err = codec.NewDecoder(f, &codec.MsgpackHandle{}).Decode(&data)
	return data, err
}
