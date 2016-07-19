package process

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	apexctx "github.com/m0sth8/context"
	"golang.org/x/net/context"

	"github.com/noxiouz/stout/isolate"
	"github.com/noxiouz/stout/pkg/semaphore"

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
	ctx          context.Context
	cancellation context.CancelFunc

	spoolPath string
	storage   codeStorage

	mu       sync.Mutex
	children map[int]*exec.Cmd
	wg       sync.WaitGroup

	spawnSm semaphore.Semaphore
}

func NewBox(ctx context.Context, cfg isolate.BoxConfig) (isolate.Box, error) {
	spoolPath, ok := cfg["spool"].(string)
	if !ok {
		spoolPath = defaultSpoolPath
	}

	var locator []string
	if endpoint, ok := cfg["locator"].(string); ok {
		locator = append(locator, endpoint)
	}

	ctx, cancel := context.WithCancel(ctx)
	box := &Box{
		ctx:          ctx,
		cancellation: cancel,

		spoolPath: spoolPath,
		storage:   createCodeStorage(locator),

		children: make(map[int]*exec.Cmd),
		// NOTE: configurable
		spawnSm: semaphore.New(10),
	}

	box.wg.Add(1)
	go func() {
		defer box.wg.Done()
		box.sigchldHandler()
	}()

	return box, nil
}

func (b *Box) Close() error {
	b.cancellation()
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
			b.mu.Lock()
			b.wait()
			b.mu.Unlock()
		case <-b.ctx.Done():
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
		// NOTE: there is possible logic race here
		// Wait -> new fork/exec replaces old one in the map -> locked.Wait
		pid, err = syscall.Wait4(-1, &ws, syscall.WNOHANG, nil)
		switch {
		case pid > 0:
			// NOTE: I fully understand that handling signals from library is a bad idea,
			// but there's nothing better in this case
			// If `pid` is not in the map, it means that it's not our worker
			// NOTE: the lock is locked in the outer scope
			pr, ok := b.children[pid]
			if ok {
				delete(b.children, pid)
				// Send SIGKILL to a process group associated with the child
				killPg(pid)
				// There is no point to check error here,
				// as it always returns "Wait error", because Wait4 has been already called.
				// But we have to call Wait to close all associated fds and to release other resources
				pr.Wait()
				procsWaitedCounter.Inc(1)
			}
		case err == syscall.EINTR:
			// NOTE: although man says that EINTR is not possible in this case, let's be on the side
			// EINTR
			// WNOHANG was not set and an unblocked signal or a SIGCHLD was caught; see signal(7).
		case err == syscall.ECHILD:
			// exec.Cmd was failed to start, but SIGCHLD arrived.
			// Actually, `non-born` child has been already waited by exec.Cmd
			// So do nothing and return
			return
		default:
			if err != nil {
				apexctx.GetLogger(b.ctx).WithError(err).Error("Wait4 error")
			}
			return
		}
	}
}

// Spawn spawns a new process
func (b *Box) Spawn(ctx context.Context, config isolate.SpawnConfig, output io.WriteCloser) (isolate.Process, error) {
	spoolPath := b.spoolPath
	if val, ok := config.Opts["spool"]; ok {
		spoolPath = fmt.Sprintf("%s", val)
	}
	workDir := filepath.Join(spoolPath, config.Name)

	var execPath = config.Executable
	if !filepath.IsAbs(config.Executable) {
		execPath = filepath.Join(workDir, config.Executable)
	}

	packedEnv := make([]string, 0, len(config.Env))
	for k, v := range config.Env {
		packedEnv = append(packedEnv, k+"="+v)
	}

	packedArgs := make([]string, 1, len(config.Args)*2+1)
	packedArgs[0] = filepath.Base(config.Executable)
	for k, v := range config.Args {
		packedArgs = append(packedArgs, k, v)
	}

	var (
		err error
		pr  *process
	)

	defer apexctx.GetLogger(ctx).WithFields(
		log.Fields{"name": config.Name, "executable": config.Executable,
			"workDir": workDir, "execPath": execPath}).Trace("processBox.Spawn").Stop(&err)

	// Update statistics
	start := time.Now()
	spawningQueueSize.Inc(1)
	if spawningQueueSize.Count() > 10 {
		spawningQueueSize.Dec(1)
		return nil, syscall.EAGAIN
	}
	err = b.spawnSm.Acquire(ctx)
	spawningQueueSize.Dec(1)
	if err != nil {
		return nil, isolate.ErrSpawningCancelled
	}
	defer b.spawnSm.Release()
	// NOTE: once process was put to the map
	// its waiter responsibility to Wait for it.

	// NOTE: No defer here
	b.mu.Lock()
	if isolate.IsCancelled(ctx) {
		b.mu.Unlock()
		return nil, isolate.ErrSpawningCancelled
	}

	newProcStart := time.Now()
	pr, err = newProcess(ctx, execPath, packedArgs, packedEnv, workDir, output)
	newProcStarted := time.Now()
	// Update has lock, so move it out from Hot spot
	defer procsNewTimer.Update(newProcStarted.Sub(newProcStart))
	if err != nil {
		b.mu.Unlock()
		procsErroredCounter.Inc(1)
		return nil, err
	}
	b.children[pr.cmd.Process.Pid] = pr.cmd
	b.mu.Unlock()

	totalSpawnTimer.UpdateSince(start)
	isolate.NotifyAbouStart(output)
	procsCreatedCounter.Inc(1)
	return pr, err
}

// Spool spools code of an app from Cocaine Storage service
func (b *Box) Spool(ctx context.Context, name string, opts isolate.Profile) (err error) {
	spoolPath := b.spoolPath
	if val, ok := opts["spool"]; ok {
		spoolPath = fmt.Sprintf("%s", val)
	}
	defer apexctx.GetLogger(ctx).WithField("name", name).WithField("spoolpath", spoolPath).Trace("processBox.Spool").Stop(&err)
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
