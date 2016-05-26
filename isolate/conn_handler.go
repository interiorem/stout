package isolate

import (
	"fmt"
	"io"
	"math/rand"
	"time"

	apexctx "github.com/m0sth8/context"
	"github.com/tinylib/msgp/msgp"
	"golang.org/x/net/context"
)

// Encoder sends replies to the Cocaine-runtime
type Encoder interface {
	Encode(interface{}) error
}

// Dispatcher handles incoming messages and keeps the state of the channel
type Dispatcher interface {
	Handle(c int64, r *msgp.Reader) (Dispatcher, error)
}

// ConnectionHandler provides method to handle accepted connection for Listener
type ConnectionHandler struct {
	ctx            context.Context
	session        map[int64]Dispatcher
	highestChannel int64

	newDispatcher dispatcherInit

	connID string
}

// NewConnectionHandler creates new ConnectionHandler
func NewConnectionHandler(ctx context.Context) (*ConnectionHandler, error) {
	// ctx = withArgsUnpacker(ctx, msgpackArgsDecoder{})
	return newConnectionHandler(ctx, newInitialDispatch)
}

func newConnectionHandler(ctx context.Context, newDisp dispatcherInit) (*ConnectionHandler, error) {
	connID := getID(ctx)
	ctx = apexctx.WithLogger(ctx, apexctx.GetLogger(ctx).WithField("conn.id", connID))

	return &ConnectionHandler{
		ctx:            ctx,
		session:        make(map[int64]Dispatcher),
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

// HandleConn decodes commands from Cocaine runtime and calls dispatchers
func (h *ConnectionHandler) HandleConn(conn io.ReadWriteCloser) {
	defer conn.Close()
	ctx, cancel := context.WithCancel(h.ctx)
	defer cancel()

	logger := apexctx.GetLogger(h.ctx)

	r := msgp.NewReader(conn)
	for {
		size, err := r.ReadArrayHeader()
		if err != nil {
			apexctx.GetLogger(h.ctx).WithError(err).Errorf("unable to read an array header")
			return
		}

		channel, err := r.ReadInt64()
		if err != nil {
			apexctx.GetLogger(h.ctx).WithError(err).Errorf("unable to read a channel")
			return
		}

		c, err := r.ReadInt64()
		if err != nil {
			apexctx.GetLogger(h.ctx).WithError(err).Errorf("unable to read a command type")
			return
		}

		// NOTE: it can be the bottleneck
		dispatcher, ok := h.session[channel]
		if !ok {
			if channel < h.highestChannel {
				apexctx.GetLogger(h.ctx).Errorf("channel has been revoked: %d %d", channel, h.highestChannel)
				return
			}

			// TODO: refactor
			dw := newDownstream(ctx, conn, channel)
			ctx = apexctx.WithLogger(ctx, logger.WithField("channel", fmt.Sprintf("%s.%d", h.connID, channel)))
			ctx = withDownstream(ctx, dw)
			dispatcher = h.newDispatcher(ctx)
		}

		dispatcher, err = dispatcher.Handle(c, r)
		if size == 4 {
			r.Skip()
		}
		// NOTE: handle ErrInvalidArgsNum
		if err != nil {
			if err == ErrInvalidArgsNum {
				apexctx.GetLogger(h.ctx).WithError(err).Errorf("protocol error")
				return
			}

			apexctx.GetLogger(h.ctx).WithError(err).Errorf("Handle returned an error")
			delete(h.session, channel)
			continue
		}
		if dispatcher == nil {
			delete(h.session, channel)
			continue
		}
		h.session[channel] = dispatcher
	}
}

type downstream struct {
	ctx     context.Context
	wr      io.Writer
	channel int64
}

func newDownstream(ctx context.Context, wr io.Writer, channel int64) Downstream {
	return &downstream{
		ctx:     ctx,
		wr:      wr,
		channel: channel,
	}
}

func (d *downstream) Reply(code int64, args ...interface{}) error {
	var (
		p   = msgpackBytePool.Get().([]byte)[:0]
		err error
	)
	defer msgpackBytePool.Put(p)

	// NOTE: `3` without headers!
	p = msgp.AppendArrayHeader(p, 3)
	p = msgp.AppendInt64(p, d.channel)
	p = msgp.AppendInt64(p, code)

	// pack args
	p = msgp.AppendArrayHeader(p, uint32(len(args)))
	for _, arg := range args {
		if p, err = msgp.AppendIntf(p, arg); err != nil {
			apexctx.GetLogger(d.ctx).WithError(err).Errorf("error dumping arg %[1]T: %[1]v", arg)
			return err
		}
	}

	if _, err = d.wr.Write(p); err != nil {
		apexctx.GetLogger(d.ctx).WithError(err).Errorf("error writing Reply")
		return err
	}

	return nil
}
