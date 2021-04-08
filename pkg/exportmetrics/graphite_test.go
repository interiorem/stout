package exportmetrics

import (
	"fmt"
	"io/ioutil"
	"net"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/context"

	_ "github.com/interiorem/stout/isolate/process"
	"github.com/rcrowley/go-metrics"
)

func TestGraphite(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	defer l.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		conn, err := l.Accept()
		if err != nil {
			t.Fatalf("l.Accept(): %v", err)
		}
		b, err := ioutil.ReadAll(conn)
		if err != nil {
			t.Fatalf("ioutil.ReadAll: %v", err)
		}
		fmt.Printf("%s", b)
	}()

	cfg := GraphiteConfig{
		Prefix:       "PREFIX.{{hostname}}",
		Addr:         l.Addr().String(),
		DurationUnit: "1s",
	}

	gr, err := NewGraphiteExporter(&cfg)
	if err != nil {
		t.Fatalf("NewGraphiteExporter: %v", err)
	}

	metrics.DefaultRegistry.Get("process_total_spawn_timer").(metrics.Timer).Update(time.Hour * 1)
	ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
	if err = gr.Send(ctx, metrics.DefaultRegistry); err != nil {
		t.Fatalf("gr.Send: %v", err)
	}

	wg.Wait()
}
