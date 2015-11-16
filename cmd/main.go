package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"

	"github.com/noxiouz/stout/server"
)

func main() {
	isolateServer, err := server.NewIsolateServer()
	if err != nil {
		log.Fatal(err)
	}

	// log.Fatal(http.ListenAndServe(":5432", isolateServer.Router))
	server := http.Server{
		Addr:     ":5432",
		Handler:  handlers.CombinedLoggingHandler(os.Stderr, isolateServer.Router),
		ErrorLog: nil,
	}
	log.Fatal(server.ListenAndServe())
}
