package process

import (
	"path/filepath"

	"golang.org/x/net/context"

	"github.com/noxiouz/stout/isolate"

	"github.com/apex/log"
)

const (
	defaultSpoolPath = "/var/spool/cocaine"
)

var (
	// can be overwritten for tests
	createCodeStorage = func(locator []string) codeStorage {
		return &cocaineCodeStorage{
			locator: locator,
		}
	}
)

type codeStorage interface {
	Spool(ctx context.Context, appname string) ([]byte, error)
}

type Box struct {
	spoolPath string
	storage   codeStorage
}

func NewBox(cfg isolate.BoxConfig) (isolate.Box, error) {
	spoolPath, ok := cfg["spool"].(string)
	if !ok {
		spoolPath = defaultSpoolPath
	}

	var locator []string
	if endpoint, ok := cfg["locator"].(string); ok {
		locator = append(locator, endpoint)
	}

	box := &Box{
		spoolPath: spoolPath,
		storage:   createCodeStorage(locator),
	}
	return box, nil
}

// Spawn spawns a new process
func (b *Box) Spawn(ctx context.Context, opts isolate.Profile, name, executable string, args, env map[string]string) (pr isolate.Process, err error) {
	workDir := filepath.Join(b.spoolPath, name)
	execPath := filepath.Join(workDir, executable)
	defer isolate.GetLogger(ctx).WithFields(log.Fields{"name": name, "executable": executable, "workDir": workDir, "execPath": execPath}).Trace("processBox.Spawn").Stop(&err)

	return newProcess(ctx, execPath, args, env, workDir)
}

// Spool spools code of an app from Cocaine Storage service
func (b *Box) Spool(ctx context.Context, name string, opts isolate.Profile) (err error) {
	defer isolate.GetLogger(ctx).WithField("name", name).Trace("processBox.Spool").Stop(&err)
	data, err := b.fetch(ctx, name)
	if err != nil {
		return err
	}

	if isolate.IsCancelled(ctx) {
		return nil
	}

	return unpackArchive(ctx, data, filepath.Join(b.spoolPath, name))
}

func (b *Box) fetch(ctx context.Context, appname string) ([]byte, error) {
	return b.storage.Spool(ctx, appname)
}
