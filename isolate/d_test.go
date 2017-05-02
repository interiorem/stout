package isolate

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/tinylib/msgp/msgp"

	"golang.org/x/net/context"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

func init() {
	Suite(&initialDispatchSuite{})
}

type testDownstreamItem struct {
	code uint64
	args []interface{}
}

type testDownstream struct {
	ch chan testDownstreamItem
}

func (t *testDownstream) Write(ctx context.Context, code uint64, data []byte) error {
	t.ch <- testDownstreamItem{code, []interface{}{data}}
	return nil
}

func (t *testDownstream) Error(ctx context.Context, code uint64, errorcode [2]int, msg string) error {
	t.ch <- testDownstreamItem{code, []interface{}{errorcode, msg}}
	return nil
}

func (t *testDownstream) Close(ctx context.Context, code uint64) error {
	t.ch <- testDownstreamItem{code, []interface{}{}}
	return nil
}

type testBox struct {
	err   error
	sleep time.Duration
}

func (b *testBox) Spool(ctx context.Context, name string, opts RawProfile) error {
	select {
	case <-ctx.Done():
		return errors.New("canceled")
	case <-time.After(b.sleep):
		return b.err
	}
}

func (b *testBox) Spawn(ctx context.Context, config SpawnConfig, wr io.Writer) (Process, error) {
	return spawnTestProcess(ctx, wr), nil
}

func (b *testBox) Inspect(ctx context.Context, workerid string) ([]byte, error) {
	return []byte("{}"), nil
}

func (b *testBox) Close() error {
	return nil
}

type testProcess struct {
	ctx    context.Context
	killed chan struct{}
}

func spawnTestProcess(ctx context.Context, wr io.Writer) *testProcess {
	pr := testProcess{
		ctx:    ctx,
		killed: make(chan struct{}),
	}

	go func() {
		var i int
		for {
			if i > 0 {
				fmt.Fprintf(wr, "output_%d\n", i)
			} else {
				fmt.Fprintf(wr, "")
			}
			i++
			select {
			case <-ctx.Done():
				return
			case <-pr.killed:
				return
			default:
				// pass
			}
		}
	}()

	return &pr
}

func (pr *testProcess) Kill() error {
	close(pr.killed)
	return nil
}

type initialDispatchSuite struct {
	ctx    context.Context
	cancel context.CancelFunc

	d       Dispatcher
	dw      *testDownstream
	session int
}

func (s *initialDispatchSuite) SetUpTest(c *C) {
	ctx, cancel := context.WithCancel(context.Background())

	boxes := Boxes{
		"testError": &testBox{err: errors.New("dummy error from testBox")},
		"testSleep": &testBox{err: nil, sleep: time.Second * 2},
		"test":      &testBox{err: nil},
	}

	ctx = context.WithValue(ctx, BoxesTag, boxes)

	s.ctx, s.cancel = ctx, cancel
	s.session = 100

	s.dw = &testDownstream{
		ch: make(chan testDownstreamItem, 1000),
	}

	d := newInitialDispatch(ctx, s.dw)
	s.d = d
}

func (s *initialDispatchSuite) TearDownTest(c *C) {
	s.cancel()
}

func (s *initialDispatchSuite) TestSpool(c *C) {
	var (
		args = map[string]interface{}{
			"type": "test",
		}
		appName     = "application"
		spoolMsg, _ = msgp.AppendIntf(nil, []interface{}{map[string]interface{}(args), appName})
	)

	// spoolDisp, err := s.d.Handle(&spoolMsg)
	spoolDisp, err := s.d.Handle(spool, msgp.NewReader(bytes.NewReader(spoolMsg)))
	c.Assert(err, IsNil)
	c.Assert(spoolDisp, FitsTypeOf, &spoolCancelationDispatch{})
	msg := <-s.dw.ch
	c.Assert(msg.code, DeepEquals, uint64(replySpoolOk))
}

func (s *initialDispatchSuite) TestSpoolCancel(c *C) {
	var (
		args = map[string]interface{}{
			"type": "testSleep",
		}
		appName      = "application"
		spoolMsg, _  = msgp.AppendIntf(nil, []interface{}{map[string]interface{}(args), appName})
		cancelMsg, _ = msgp.AppendIntf(nil, []interface{}{})
	)

	spoolDisp, err := s.d.Handle(spool, msgp.NewReader(bytes.NewReader(spoolMsg)))
	c.Assert(err, IsNil)
	c.Assert(spoolDisp, FitsTypeOf, &spoolCancelationDispatch{})
	spoolDisp.Handle(spoolCancel, msgp.NewReader(bytes.NewReader(cancelMsg)))
	msg := <-s.dw.ch
	c.Assert(msg.code, DeepEquals, uint64(replySpoolOk))
}

func (s *initialDispatchSuite) TestSpoolError(c *C) {
	var (
		args = map[string]interface{}{
			"type": "testError",
		}
		appName     = "application"
		spoolMsg, _ = msgp.AppendIntf(nil, []interface{}{map[string]interface{}(args), appName})
	)

	spoolDisp, err := s.d.Handle(spool, msgp.NewReader(bytes.NewReader(spoolMsg)))
	c.Assert(err, IsNil)
	c.Assert(spoolDisp, FitsTypeOf, &spoolCancelationDispatch{})
	msg := <-s.dw.ch
	c.Assert(msg.code, Equals, uint64(replySpoolError))
}

func (s *initialDispatchSuite) TestSpawnAndKill(c *C) {
	var (
		opts = map[string]interface{}{
			"type": "testSleep",
		}
		appName    = "application"
		executable = "test_app.exe"
		args       = make(map[string]string, 0)
		env        = make(map[string]string, 0)
		// spawnMsg   = message{s.session, spawn, []interface{}{opts, appName, executable, args, env}}
		spawnMsg, _ = msgp.AppendIntf(nil, []interface{}{map[string]interface{}(opts), appName, executable, args, env})
		killMsg, _  = msgp.AppendIntf(nil, []interface{}{})
	)
	spawnDisp, err := s.d.Handle(spawn, msgp.NewReader(bytes.NewReader(spawnMsg)))
	c.Assert(err, IsNil)
	c.Assert(spawnDisp, FitsTypeOf, &spawnDispatch{})

	// First chunk must be empty to notify about start
	msg := <-s.dw.ch
	c.Assert(msg.code, DeepEquals, uint64(replySpawnWrite))
	c.Assert(msg.args, HasLen, 1)
	data, ok := msg.args[0].([]byte)
	c.Assert(ok, Equals, true)
	c.Assert(data, HasLen, 0)

	// Let's read some output
	msg = <-s.dw.ch
	c.Assert(msg.code, Equals, uint64(replySpawnWrite))
	c.Assert(msg.args, HasLen, 1)

	data, ok = msg.args[0].([]byte)
	c.Assert(ok, Equals, true)
	c.Assert(data, Not(HasLen), 0)

	noneDisp, err := spawnDisp.Handle(spawnKill, msgp.NewReader(bytes.NewReader(killMsg)))
	c.Assert(err, IsNil)
	c.Assert(noneDisp, IsNil)
}
