package isolate

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	apexctx "github.com/m0sth8/context"
	"github.com/tinylib/msgp/msgp"
	"golang.org/x/net/context"
)

type sessions struct {
	sync.Mutex

	session map[int64]Dispatcher
}

func newSessions() *sessions {
	return &sessions{
		session: make(map[int64]Dispatcher),
	}
}

func (s *sessions) Attach(channel int64, dispatch Dispatcher) {
	s.Lock()
	s.session[channel] = dispatch
	s.Unlock()
}

func (s *sessions) Detach(channel int64) {
	s.Lock()
	delete(s.session, channel)
	s.Unlock()
}

func (s *sessions) Get(channel int64) (Dispatcher, bool) {
	s.Lock()
	dispatch, ok := s.session[channel]
	s.Unlock()
	return dispatch, ok
}

// Dispatcher handles incoming messages and keeps the state of the channel
type Dispatcher interface {
	Handle(c int64, r *msgp.Reader) (Dispatcher, error)
}

// ConnectionHandler provides method to handle accepted connection for Listener
type ConnectionHandler struct {
	ctx context.Context
	*sessions
	highestChannel int64

	newDispatcher dispatcherInit

	connID string
}

// NewConnectionHandler creates new ConnectionHandler
func NewConnectionHandler(ctx context.Context) (*ConnectionHandler, error) {
	return newConnectionHandler(ctx, newInitialDispatch)
}

func newConnectionHandler(ctx context.Context, newDisp dispatcherInit) (*ConnectionHandler, error) {
	connID := getID(ctx)
	ctx = apexctx.WithLogger(ctx, apexctx.GetLogger(ctx).WithField("conn.id", connID))

	return &ConnectionHandler{
		ctx:            ctx,
		sessions:       newSessions(),
		highestChannel: 0,

		newDispatcher: newDisp,

		connID: connID,
	}, nil
}

func getID(ctx context.Context) string {
	var uniqueid string
	uniqueid, ok := ctx.Value("conn.id").(string)
	if !ok {
		return fmt.Sprintf("%d.%d", time.Now().Unix(), rand.Int63())
	}

	return uniqueid
}

func (h *ConnectionHandler) next(r *msgp.Reader) (sz uint32, channel int64, c int64, err error) {
	sz, err = r.ReadArrayHeader()
	if err != nil {
		return
	}

	channel, err = r.ReadInt64()
	if err != nil {
		return
	}

	c, err = r.ReadInt64()
	if err != nil {
		return
	}

	return
}

// HandleConn decodes commands from Cocaine runtime and calls dispatchers
func (h *ConnectionHandler) HandleConn(conn io.ReadWriteCloser) {
	defer conn.Close()
	ctx, cancel := context.WithCancel(h.ctx)
	defer cancel()
	logger := apexctx.GetLogger(h.ctx)

	r := msgp.NewReader(conn)
	for {
		size, channel, c, err := h.next(r)
		if err != nil {
			if err == io.EOF {
				apexctx.GetLogger(h.ctx).Errorf("Connection has been closed")
				return
			}
			apexctx.GetLogger(h.ctx).WithError(err).Errorf("next(): unable to read message")
			return
		}
		logger.Infof("array length %d, channel %d, number %d", size, channel, c)

		// NOTE: it can be the bottleneck
		dispatcher, ok := h.sessions.Get(channel)
		if !ok {
			if channel < h.highestChannel {
				logger.Errorf("channel has been revoked: %d %d", channel, h.highestChannel)
				return
			} else if channel == h.highestChannel {
				// NOTE: we cannot reply to this, because Downstream has been already closed
				return
			}

			h.highestChannel = channel

			ctx = apexctx.WithLogger(ctx, logger.WithField("channel", fmt.Sprintf("%s.%d", h.connID, channel)))
			rs := newResponseStream(ctx, conn, channel)
			rs.OnClose(func(ctx context.Context) {
				h.sessions.Detach(channel)
			})
			dispatcher = h.newDispatcher(ctx, rs)
		}

		dispatcher, err = dispatcher.Handle(c, r)
		// NOTE: remove it when the headers are being handling properly
		if size == 4 {
			r.Skip()
		}

		if err != nil {
			if err == ErrInvalidArgsNum {
				logger.WithError(err).Errorf("channel %d, number %d", channel, c)
				return
			}

			logger.WithError(err).Errorf("Handle returned an error")
			h.sessions.Detach(channel)
			continue
		}
		if dispatcher == nil {
			h.sessions.Detach(channel)
			continue
		}

		h.sessions.Attach(channel, dispatcher)
	}
}

type responseStream struct {
	ctx     context.Context
	wr      io.Writer
	channel int64

	onClose func(ctx context.Context)
	closed  uint32
}

var errStreamIsClosed = errors.New("Stream is closed")

func newResponseStream(ctx context.Context, wr io.Writer, channel int64) *responseStream {
	return &responseStream{
		ctx:     ctx,
		wr:      wr,
		channel: channel,
		onClose: nil,
	}
}

func (r *responseStream) OnClose(onClose func(context.Context)) {
	r.onClose = onClose
}

func (r *responseStream) close(ctx context.Context) error {
	if !atomic.CompareAndSwapUint32(&r.closed, 0, 1) {
		return errStreamIsClosed
	}

	if r.onClose != nil {
		r.onClose(ctx)
	}

	return nil
}

func (r *responseStream) Write(ctx context.Context, num int64, data []byte) error {
	if atomic.LoadUint32(&r.closed) == 1 {
		apexctx.GetLogger(r.ctx).WithError(errStreamIsClosed).Error("responseStream.Write")
		return errStreamIsClosed
	}

	p := msgpackBytePool.Get().([]byte)[:0]
	defer msgpackBytePool.Put(p)

	// NOTE: `3` without headers!
	p = msgp.AppendArrayHeader(p, 3)
	p = msgp.AppendInt64(p, r.channel)
	p = msgp.AppendInt64(p, num)

	p = msgp.AppendArrayHeader(p, 1)
	p = msgp.AppendStringFromBytes(p, data)

	if _, err := r.wr.Write(p); err != nil {
		apexctx.GetLogger(r.ctx).WithError(err).Error("responseStream.Write")
		return err
	}
	return nil
}

func (r *responseStream) Error(ctx context.Context, num int64, code [2]int, msg string) error {
	if err := r.close(ctx); err != nil {
		apexctx.GetLogger(r.ctx).WithError(err).Error("responseStream.Error")
		return err
	}

	p := msgpackBytePool.Get().([]byte)[:0]
	defer msgpackBytePool.Put(p)

	// NOTE: `3` without headers!
	p = msgp.AppendArrayHeader(p, 3)
	p = msgp.AppendInt64(p, r.channel)
	p = msgp.AppendInt64(p, num)

	// code_category + error message
	p = msgp.AppendArrayHeader(p, 2)

	// code & category
	p = msgp.AppendArrayHeader(p, 2)
	p = msgp.AppendInt(p, code[0])
	p = msgp.AppendInt(p, code[1])

	// error message
	p = msgp.AppendString(p, msg)

	if _, err := r.wr.Write(p); err != nil {
		apexctx.GetLogger(r.ctx).WithError(err).Errorf("responseStream.Error")
		return err
	}
	return nil
}

func (r *responseStream) Close(ctx context.Context, num int64) error {
	if err := r.close(ctx); err != nil {
		apexctx.GetLogger(r.ctx).WithError(err).Error("responseStream.Close")
		return err
	}

	if r.onClose != nil {
		r.onClose(ctx)
	}

	p := msgpackBytePool.Get().([]byte)[:0]
	defer msgpackBytePool.Put(p)

	// NOTE: `3` without headers!
	p = msgp.AppendArrayHeader(p, 3)
	p = msgp.AppendInt64(p, r.channel)
	p = msgp.AppendInt64(p, num)

	p = msgp.AppendArrayHeader(p, 0)

	if _, err := r.wr.Write(p); err != nil {
		apexctx.GetLogger(r.ctx).WithError(err).Errorf("responseStream.Error")
		return err
	}
	return nil
}
