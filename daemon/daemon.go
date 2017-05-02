package daemon

import (
	"crypto/md5"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	apexlog "github.com/apex/log"
	"golang.org/x/net/context"

	"github.com/noxiouz/stout/isolate"
	"github.com/noxiouz/stout/pkg/config"
	"github.com/noxiouz/stout/pkg/log"
)

type Daemon struct {
	boxes     isolate.Boxes
	cfg       *config.Config
	listeners []net.Listener

	muListeners sync.Mutex
}

func New(ctx context.Context, configuration *config.Config) (*Daemon, error) {
	checkLimits(ctx)
	d := Daemon{
		cfg:       configuration,
		boxes:     make(isolate.Boxes),
		listeners: make([]net.Listener, 0),
	}

	boxTypes := map[string]struct{}{}
	for name, cfg := range configuration.Isolate {
		if _, ok := boxTypes[cfg.Type]; ok {
			log.G(ctx).WithField("box", name).WithField("type", cfg.Type).Error("dublicated box type")
			d.Close()
			return nil, fmt.Errorf("duplicated box type %s", cfg.Type)
		}
		boxTypes[cfg.Type] = struct{}{}

		boxCtx := log.WithLogger(ctx, log.G(ctx).WithField("box", name))
		box, err := isolate.ConstructBox(boxCtx, cfg.Type, cfg.Args)
		if err != nil {
			log.G(ctx).WithError(err).WithField("box", name).WithField("type", cfg.Type).Error("unable to create box")
			d.Close()
			return nil, err
		}
		d.boxes[name] = box
	}

	return &d, nil
}

func (d *Daemon) RegisterHTTPHandlers(ctx context.Context, mux *http.ServeMux) {
	for name := range d.boxes {
		http.HandleFunc("/inspect/"+name, func(name string) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case "GET", "HEAD":
				default:
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				box, ok := d.boxes[name]
				if !ok {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintf(w, "Box %s is unavailable", name)
					return
				}

				workerid := r.URL.Query().Get("uuid")
				if workerid == "" {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintln(w, "query arg uuid is not set")
					return
				}

				data, err := box.Inspect(ctx, workerid)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintf(w, "Box.Inspect %s failed %v\n", name, err)
					return
				}

				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
				w.Write(data)
			}
		}(name))
	}
}

func (d *Daemon) Serve(ctx context.Context) error {
	var listeners = make([]net.Listener, 0, len(d.cfg.Endpoints))
	for _, endpoint := range d.cfg.Endpoints {
		log.G(ctx).WithField("endpoint", endpoint).Info("start TCP server")
		ln, err := net.Listen("tcp", endpoint)
		if err != nil {
			log.G(ctx).WithError(err).WithField("endpoint", endpoint).Error("unable to listen to")
			closeListeners(listeners)
			return err
		}
		listeners = append(listeners, ln)
	}

	return d.ServeOnListeners(ctx, listeners)
}

func (d *Daemon) ServeOnListeners(ctx context.Context, listeners []net.Listener) error {
	d.closeListeners()

	d.muListeners.Lock()
	d.listeners = listeners
	d.muListeners.Unlock()

	ctx, cancelFunc := context.WithCancel(context.WithValue(ctx, isolate.BoxesTag, d.boxes))
	defer cancelFunc()

	var wg sync.WaitGroup
	for _, ln := range listeners {
		wg.Add(1)
		go func(ln net.Listener) {
			defer wg.Done()
			lnLogger := log.G(ctx).WithField("listener", ln.Addr())
			for {
				conn, err := ln.Accept()
				if err != nil {
					lnLogger.WithError(err).Error("Accept")
					return
				}

				// TODO: more optimal way
				connID := fmt.Sprintf("%.3x", md5.Sum([]byte(fmt.Sprintf("%s.%d", conn.RemoteAddr().String(), time.Now().Unix()))))
				lnLogger.WithFields(apexlog.Fields{"remote.addr": conn.RemoteAddr(), "conn.id": connID}).Info("accepted new connection")

				go func() {
					conns.Inc(0)
					defer conns.Dec(0)
					isolate.NewConnectionHandler(context.WithValue(ctx, "conn.id", connID)).HandleConn(conn)
				}()
			}
		}(ln)
	}

	wg.Wait()
	return nil
}

func closeListeners(listeners []net.Listener) {
	for _, ln := range listeners {
		ln.Close()
	}
}

func (d *Daemon) closeListeners() {
	d.muListeners.Lock()
	defer d.muListeners.Unlock()
	closeListeners(d.listeners)
}

func (d *Daemon) closeBoxes() {
	for _, box := range d.boxes {
		box.Close()
	}
}

func (d *Daemon) Close() {
	d.closeListeners()
	d.closeBoxes()
}
