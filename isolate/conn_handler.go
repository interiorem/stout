package isolate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"

	"github.com/tinylib/msgp/msgp"
	"github.com/uber-go/zap"

	"github.com/noxiouz/stout/pkg/log"
)

type sessions struct {
	sync.Mutex

	session map[uint64]Dispatcher
}

func newSessions() *sessions {
	return &sessions{
		session: make(map[uint64]Dispatcher),
	}
}

func (s *sessions) Attach(channel uint64, dispatch Dispatcher) {
	s.Lock()
	s.session[channel] = dispatch
	s.Unlock()
}

func (s *sessions) Detach(channel uint64) {
	s.Lock()
	delete(s.session, channel)
	s.Unlock()
}

func (s *sessions) Get(channel uint64) (Dispatcher, bool) {
	s.Lock()
	dispatch, ok := s.session[channel]
	s.Unlock()
	return dispatch, ok
}

// Dispatcher handles incoming messages and keeps the state of the channel
type Dispatcher interface {
	Handle(c uint64, r *msgp.Reader) (Dispatcher, error)
}

// ConnectionHandler provides method to handle accepted connection for Listener
type ConnectionHandler struct {
	ctx context.Context
	*sessions
	highestChannel uint64

	newDispatcher dispatcherInit

	connID string
}

// NewConnectionHandler creates new ConnectionHandler
func NewConnectionHandler(ctx context.Context) (*ConnectionHandler, error) {
	return newConnectionHandler(ctx, newInitialDispatch)
}

func newConnectionHandler(ctx context.Context, newDisp dispatcherInit) (*ConnectionHandler, error) {
	connID := getID(ctx)
	ctx = log.WithLogger(ctx, log.G(ctx).With(zap.String("conn.id", connID)))

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

func (h *ConnectionHandler) next(r *msgp.Reader) (hasHeaders bool, channel uint64, c uint64, err error) {
	var sz uint32
	sz, err = r.ReadArrayHeader()
	if err != nil {
		return
	}
	hasHeaders = sz == 4

	channel, err = r.ReadUint64()
	if err != nil {
		return
	}

	c, err = r.ReadUint64()
	if err != nil {
		return
	}

	return
}

// HandleConn decodes commands from Cocaine runtime and calls dispatchers
func (h *ConnectionHandler) HandleConn(conn io.ReadWriteCloser) {
	defer func() {
		conn.Close()
		log.G(h.ctx).Error("Connection has been closed")
	}()

	ctx, cancel := context.WithCancel(h.ctx)
	defer cancel()
	logger := log.G(h.ctx)

	r := msgp.NewReader(conn)
LOOP:
	for {
		hasHeaders, channel, c, err := h.next(r)
		if err != nil {
			if err == io.EOF {
				return
			}
			log.G(h.ctx).Error("next(): unable to read message", zap.Error(err))
			return
		}
		logger.Info(fmt.Sprintf("channel %d, number %d", channel, c))

		dispatcher, ok := h.sessions.Get(channel)
		if !ok {
			if channel <= h.highestChannel {
				// dispatcher was detached from ResponseStream.OnClose
				// This message must be `close` message.
				// `channel`, `number` are parsed, skip `args` and probably `headers`
				logger.Info("dispatcher was detached", zap.Uint64("channel", channel))
				r.Skip()
				if hasHeaders {
					r.Skip()
				}
				continue LOOP
			}

			h.highestChannel = channel

			ctx = log.WithLogger(ctx, logger.With(zap.String("channel", fmt.Sprintf("%s.%d", h.connID, channel))))
			rs := newResponseStream(ctx, conn, channel)
			rs.OnClose(func(ctx context.Context) {
				h.sessions.Detach(channel)
			})
			dispatcher = h.newDispatcher(ctx, rs)
		}

		dispatcher, err = dispatcher.Handle(c, r)
		// NOTE: remove it when the headers are being handling properly
		if hasHeaders {
			r.Skip()
		}

		if err != nil {
			if err == ErrInvalidArgsNum {
				logger.Error("Exit from Handle", zap.Error(err), zap.Uint64("channel", channel), zap.Uint64("number", c))
				return
			}

			logger.Error("Handle returned an error", zap.Error(err))
			h.sessions.Detach(channel)
			continue LOOP
		}
		if dispatcher == nil {
			h.sessions.Detach(channel)
			continue LOOP
		}

		h.sessions.Attach(channel, dispatcher)
	}
}

type responseStream struct {
	sync.Mutex

	ctx     context.Context
	wr      io.Writer
	channel uint64

	onClose func(ctx context.Context)
	closed  bool
}

var errStreamIsClosed = errors.New("Stream is closed")

func newResponseStream(ctx context.Context, wr io.Writer, channel uint64) *responseStream {
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

func (r *responseStream) close(ctx context.Context) {
	if r.closed {
		return
	}

	if r.onClose != nil {
		r.onClose(ctx)
	}
}

func (r *responseStream) Write(ctx context.Context, num uint64, data []byte) error {
	r.Lock()
	defer r.Unlock()

	if r.closed {
		log.G(r.ctx).Error("responseStream.Write", zap.Error(errStreamIsClosed))
		return errStreamIsClosed
	}

	p := msgpackBytePool.Get().([]byte)[:0]
	defer msgpackBytePool.Put(p)

	// NOTE: `3` without headers!
	p = msgp.AppendArrayHeader(p, 3)
	p = msgp.AppendUint64(p, r.channel)
	p = msgp.AppendUint64(p, num)

	p = msgp.AppendArrayHeader(p, 1)
	p = msgp.AppendStringFromBytes(p, data)

	if _, err := r.wr.Write(p); err != nil {
		log.G(r.ctx).Error("responseStream.Write", zap.Error(err))
		return err
	}
	return nil
}

func (r *responseStream) Error(ctx context.Context, num uint64, code [2]int, msg string) error {
	r.Lock()
	defer r.Unlock()
	if r.closed {
		log.G(r.ctx).Error("responseStream.Error", zap.Error(errStreamIsClosed))
		return errStreamIsClosed
	}
	defer r.close(ctx)

	p := msgpackBytePool.Get().([]byte)[:0]
	defer msgpackBytePool.Put(p)

	// NOTE: `3` without headers!
	p = msgp.AppendArrayHeader(p, 3)
	p = msgp.AppendUint64(p, r.channel)
	p = msgp.AppendUint64(p, num)

	// code_category + error message
	p = msgp.AppendArrayHeader(p, 2)

	// code & category
	p = msgp.AppendArrayHeader(p, 2)
	p = msgp.AppendInt(p, code[0])
	p = msgp.AppendInt(p, code[1])

	// error message
	p = msgp.AppendString(p, msg)

	if _, err := r.wr.Write(p); err != nil {
		log.G(r.ctx).Error("responseStream.Error", zap.Error(err))
		return err
	}
	return nil
}

func (r *responseStream) Close(ctx context.Context, num uint64) error {
	r.Lock()
	defer r.Unlock()
	if r.closed {
		log.G(r.ctx).Error("responseStream.Close", zap.Error(errStreamIsClosed))
		return errStreamIsClosed
	}
	defer r.close(ctx)

	p := msgpackBytePool.Get().([]byte)[:0]
	defer msgpackBytePool.Put(p)

	// NOTE: `3` without headers!
	p = msgp.AppendArrayHeader(p, 3)
	p = msgp.AppendUint64(p, r.channel)
	p = msgp.AppendUint64(p, num)

	p = msgp.AppendArrayHeader(p, 0)
	if _, err := r.wr.Write(p); err != nil {
		log.G(r.ctx).Error("responseStream.Error", zap.Error(err))
		return err
	}
	return nil
}
