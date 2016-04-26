package isolate

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"golang.org/x/net/context"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

func init() {
	Suite(&initialDispatchSuite{})
}

type testDownstreamItem struct {
	code int
	args []interface{}
}

type testDownstream struct {
	ch chan testDownstreamItem
}

func (t *testDownstream) Reply(code int, args ...interface{}) error {
	t.ch <- testDownstreamItem{code, args}
	return nil
}

type testBox struct {
	err   error
	sleep time.Duration
}

func (b *testBox) Spool(ctx context.Context, name string, opts Profile) error {
	select {
	case <-ctx.Done():
		return errors.New("canceled")
	case <-time.After(b.sleep):
		return b.err
	}
}

func (b *testBox) Spawn(ctx context.Context, opts Profile, name, executable string, args, env map[string]string) (Process, error) {
	return spawnTestProcess(ctx), nil
}

func (b *testBox) Close() error {
	return nil
}

type testProcess struct {
	ctx    context.Context
	killed chan struct{}
	output chan ProcessOutput
}

func spawnTestProcess(ctx context.Context) *testProcess {
	pr := testProcess{
		ctx:    ctx,
		killed: make(chan struct{}),
		output: make(chan ProcessOutput),
	}

	go func() {
		var i int
		for {
			var out = ProcessOutput{Data: []byte("")}
			if i > 0 {
				out.Data = []byte(fmt.Sprintf("output_%d\n", i))
			}
			i++
			select {
			case pr.output <- out:
			case <-pr.killed:
				return
			case <-pr.ctx.Done():
				return
			}
		}
	}()

	return &pr
}

func (pr *testProcess) Output() <-chan ProcessOutput {
	return pr.output
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

	ctx = withArgsUnpacker(ctx, msgpackArgsDecoder{})
	ctx = context.WithValue(ctx, BoxesTag, boxes)

	s.ctx, s.cancel = ctx, cancel
	s.session = 100

	s.dw = &testDownstream{
		ch: make(chan testDownstreamItem),
	}

	ctx = withDownstream(s.ctx, s.dw)
	d := newInitialDispatch(ctx)
	s.d = d
}

func (s *initialDispatchSuite) TearDownTest(c *C) {
	s.cancel()
}

func (s *initialDispatchSuite) TestSpool(c *C) {
	var (
		args = Profile{
			"type": "test",
		}
		appName  = "application"
		spoolMsg = message{s.session, spool, []interface{}{args, appName}}
	)

	spoolDisp, err := s.d.Handle(&spoolMsg)
	c.Assert(err, IsNil)
	c.Assert(spoolDisp, FitsTypeOf, &spoolCancelationDispatch{})
	msg := <-s.dw.ch
	c.Assert(msg.code, Equals, replySpoolOk)
}

func (s *initialDispatchSuite) TestSpoolCancel(c *C) {
	var (
		args = Profile{
			"type": "testSleep",
		}
		appName   = "application"
		spoolMsg  = message{s.session, spool, []interface{}{args, appName}}
		cancelMsg = message{s.session, spoolCancel, []interface{}{}}
	)

	spoolDisp, err := s.d.Handle(&spoolMsg)
	c.Assert(err, IsNil)
	c.Assert(spoolDisp, FitsTypeOf, &spoolCancelationDispatch{})
	spoolDisp.Handle(&cancelMsg)
	msg := <-s.dw.ch
	c.Assert(msg.code, Equals, replySpoolError)
}

func (s *initialDispatchSuite) TestSpoolError(c *C) {
	var (
		args = Profile{
			"type": "testError",
		}
		appName  = "application"
		spoolMsg = message{s.session, spool, []interface{}{args, appName}}
	)

	spoolDisp, err := s.d.Handle(&spoolMsg)
	c.Assert(err, IsNil)
	c.Assert(spoolDisp, FitsTypeOf, &spoolCancelationDispatch{})
	msg := <-s.dw.ch
	c.Assert(msg.code, Equals, replySpoolError)
}

func (s *initialDispatchSuite) TestSpawnAndKill(c *C) {
	var (
		opts = Profile{
			"type": "testSleep",
		}
		appName    = "application"
		executable = "test_app.exe"
		args       = make(map[string]string, 0)
		env        = make(map[string]string, 0)
		spawnMsg   = message{s.session, spawn, []interface{}{opts, appName, executable, args, env}}
		killMsg    = message{s.session, spawnKill, []interface{}{}}
	)
	spawnDisp, err := s.d.Handle(&spawnMsg)
	c.Assert(err, IsNil)
	c.Assert(spawnDisp, FitsTypeOf, &spawnDispatch{})

	// First chunk must be empty to notify about start
	msg := <-s.dw.ch
	c.Assert(msg.code, Equals, replySpawnWrite)
	c.Assert(msg.args, HasLen, 1)
	data, ok := msg.args[0].([]byte)
	c.Assert(ok, Equals, true)
	c.Assert(data, HasLen, 0)

	// Let's read some output
	msg = <-s.dw.ch
	c.Assert(msg.code, Equals, replySpawnWrite)
	c.Assert(msg.args, HasLen, 1)

	data, ok = msg.args[0].([]byte)
	c.Assert(ok, Equals, true)
	c.Assert(data, Not(HasLen), 0)

	noneDisp, err := spawnDisp.Handle(&killMsg)
	c.Assert(err, IsNil)
	c.Assert(noneDisp, IsNil)
}
