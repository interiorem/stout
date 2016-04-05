package process

import (
	"path/filepath"

	"golang.org/x/net/context"

	"github.com/noxiouz/stout/isolation"

	"github.com/apex/log"
)

const (
	defaultSpoolPath = "/var/spool/cocaine"
)

var (
	// can be overwritten for tests
	createCodeStorage = func() codeStorage {
		return &cocaineCodeStorage{}
	}
)

type codeStorage interface {
	Spool(ctx context.Context, appname string) ([]byte, error)
}

type Box struct {
	spoolPath string
	storage   codeStorage
}

func NewBox(cfg isolation.BoxConfig) (isolation.Box, error) {
	spoolPath, ok := cfg["spool"].(string)
	if !ok {
		spoolPath = defaultSpoolPath
	}

	box := &Box{
		spoolPath: spoolPath,
		storage:   createCodeStorage(),
	}
	return box, nil
}

// Spawn spawns a new process
func (b *Box) Spawn(ctx context.Context, opts isolation.Profile, name, executable string, args, env map[string]string) (pr isolation.Process, err error) {
	workDir := filepath.Join(b.spoolPath, name)
	execPath := filepath.Join(workDir, executable)
	defer isolation.GetLogger(ctx).Trace("processBox.Spawn").WithFields(log.Fields{"name": name, "executable": executable, "workDir": workDir, "execPath": execPath}).Stop(&err)

	return newProcess(ctx, execPath, args, env, workDir)
}

// Spool spools code of an app from Cocaine Storage service
func (b *Box) Spool(ctx context.Context, name string, opts isolation.Profile) (err error) {
	defer isolation.GetLogger(ctx).Trace("processBox.Spool").WithField("name", name).Stop(&err)
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
	return b.storage.Spool(ctx, appname)
}
