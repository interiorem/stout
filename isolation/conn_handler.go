package isolation

import (
	"io"
	"log"

	"golang.org/x/net/context"
)

type Decoder interface {
	Decode(interface{}) error
}

type message struct {
	Channel int
	Number  int
	Args    []interface{}
}

type Dispatcher interface {
	Handle(msg *message) (Dispatcher, error)
}

type connectionHandler struct {
	ctx            context.Context
	session        map[int]Dispatcher
	highestChannel int

	newDecoder    decoderInit
	newDispatcher dispatcherInit
}

func newConnectionHandler(ctx context.Context, newDec decoderInit, newDisp dispatcherInit) (*connectionHandler, error) {
	return &connectionHandler{
		ctx:            ctx,
		session:        make(map[int]Dispatcher),
		highestChannel: 0,

		newDecoder:    newDec,
		newDispatcher: newDisp,
	}, nil
}

func (h *connectionHandler) handleConn(conn io.ReadWriteCloser) error {
	defer conn.Close()
	var decoder Decoder = h.newDecoder(conn)
LOOP:
	for {
		var (
			msg message
			err error
		)

		if err := decoder.Decode(&msg); err != nil {
			return err
		}

		dispatcher, ok := h.session[msg.Channel]
		if !ok {
			if msg.Number < h.highestChannel {
				log.Printf("corrupted channel order: %d %d", msg.Number, h.highestChannel)
				continue LOOP
			}

			dispatcher, err = h.newDispatcher(h.ctx)
			if err != nil {
				log.Fatalf("unable to create initial dispatch: %v", err)
			}
		}
		log.Println(dispatcher)

		dispatcher, err = dispatcher.Handle(&msg)
		if err != nil {
			log.Printf("dispatch.Handler returns error: %v", err)
			continue LOOP
		}
		h.session[msg.Channel] = dispatcher
	}
}
