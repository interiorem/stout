package process

import (
	"expvar"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

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

var (
	boxStat = expvar.NewMap("process.box")
)

type codeStorage interface {
	Spool(ctx context.Context, appname string) ([]byte, error)
}

type Box struct {
	spoolPath string
	storage   codeStorage

	mu       sync.Mutex
	children map[int]*exec.Cmd
	wg       sync.WaitGroup

	onClose chan struct{}
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

		children: make(map[int]*exec.Cmd),

		onClose: make(chan struct{}),
	}

	box.wg.Add(1)
	go func() {
		defer box.wg.Done()
		box.sigchldHandler()
	}()

	boxStat.Set("processes", expvar.Func(func() interface{} {
		box.mu.Lock()
		defer box.mu.Unlock()
		pids := make([]int, 0, len(box.children))
		for k := range box.children {
			pids = append(pids, k)
		}
		return pids
	}))

	return box, nil
}

func (b *Box) Close() error {
	close(b.onClose)
	b.wg.Wait()
	return nil
}

func (b *Box) sigchldHandler() {
	sigchld := make(chan os.Signal, 1)
	signal.Notify(sigchld, syscall.SIGCHLD)
	defer signal.Stop(sigchld)

	for {
		select {
		case <-sigchld:
			// NOTE: due to possible signal merging
			// Box.wait tries to call Wait unless ECHILD occures
			b.wait()
		case <-b.onClose:
			return
		}
	}
}

func (b *Box) wait() {
	var (
		ws  syscall.WaitStatus
		pid int
		err error
	)

	for {
		switch pid, err = syscall.Wait4(-1, &ws, syscall.WNOHANG, nil); err {
		case syscall.EINTR:
			// pass to retry again
		case syscall.ECHILD:
			return
		case nil:
			// NOTE: I fully understand that handling signals from library is a bad idea,
			// but there's nothing better in this case
			b.mu.Lock()
			// If `pid` is not in the map, it means that it's not our worker
			pr, ok := b.children[pid]
			if ok {
				delete(b.children, pid)
			}
			b.mu.Unlock()
			if ok {
				// There is no point to check error here,
				// as it always returns "Wait error", because Wait4 has been already called.
				// But we have to call Wait to close all associated fds and to release other resources
				pr.Wait()
				boxStat.Add("waited", 1)
			}
		default:
			fmt.Printf("Wait4 error: %v\n", err)
			return
		}
	}
}

// Spawn spawns a new process
func (b *Box) Spawn(ctx context.Context, opts isolate.Profile, name, executable string, args, env map[string]string) (isolate.Process, error) {
	spoolPath := b.spoolPath
	if val, ok := opts["spool"]; ok {
		spoolPath = fmt.Sprintf("%s", val)
	}
	workDir := filepath.Join(spoolPath, name)

	var execPath = executable
	if !filepath.IsAbs(executable) {
		execPath = filepath.Join(workDir, executable)
	}

	var (
		err error
		pr  *process
	)

	defer isolate.GetLogger(ctx).WithFields(
		log.Fields{"name": name, "executable": executable,
			"workDir": workDir, "execPath": execPath}).Trace("processBox.Spawn").Stop(&err)

	// NOTE: once process was put to the map
	// its waiter responsibility to Wait for it.
	boxStat.Add("spawned", 1)
	b.mu.Lock()
	defer b.mu.Unlock()
	pr, err = newProcess(ctx, execPath, args, env, workDir)
	if err != nil {
		boxStat.Add("crashed", 1)
		return nil, err
	}
	b.children[pr.cmd.Process.Pid] = pr.cmd

	return pr, err
}

// Spool spools code of an app from Cocaine Storage service
func (b *Box) Spool(ctx context.Context, name string, opts isolate.Profile) (err error) {
	spoolPath := b.spoolPath
	if val, ok := opts["spool"]; ok {
		spoolPath = fmt.Sprintf("%s", val)
	}
	defer isolate.GetLogger(ctx).WithField("name", name).WithField("spoolpath", spoolPath).Trace("processBox.Spool").Stop(&err)
	data, err := b.fetch(ctx, name)
	if err != nil {
		return err
	}

	if isolate.IsCancelled(ctx) {
		return nil
	}

	return unpackArchive(ctx, data, filepath.Join(spoolPath, name))
}

func (b *Box) fetch(ctx context.Context, appname string) ([]byte, error) {
	return b.storage.Spool(ctx, appname)
}
