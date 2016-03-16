package main

import (
	"flag"
	"log"
	"net"

	"golang.org/x/net/context"

	"github.com/noxiouz/stout/isolation"
	"github.com/noxiouz/stout/isolation/process"
)

var (
	endpoint string
)

func init() {
	flag.StringVar(&endpoint, "endpoint", ":29042", "endpoint like 0.0.0.0:29042")
	flag.Parse()
}

func main() {
	log.Printf("listening to %s", endpoint)

	box, err := process.NewBox(isolation.BoxConfig{
		"fileStorage": "/tmp/cocaine",
	})

	if err != nil {
		log.Fatalf("unable to create box: %v", err)
	}

	boxes := isolation.Boxes{
		"process": box,
	}

	ctx := context.WithValue(context.Background(), isolation.BoxesTag, boxes)
	ln, err := net.Listen("tcp", endpoint)
	if err != nil {
		log.Fatalf("unable to listen to %s: %v", endpoint, err)
	}

	for {
		conn, err := ln.Accept()
		log.Printf("accepted connection from %s", conn.RemoteAddr())
		if err != nil {
			log.Printf("acception error: %v", err)
			continue
		}
		connHandler, err := isolation.NewConnectionHandler(ctx)
		if err != nil {
			log.Fatalf("unbale to create connHandler: %v", err)
		}

		go connHandler.HandleConn(conn)
	}
}
