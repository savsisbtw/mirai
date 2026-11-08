package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8787", "listen address")
	dataDir := flag.String("data", "data", "data directory")
	flag.Parse()

	if err := os.MkdirAll(*dataDir, 0o755); err != nil {
		log.Fatal(err)
	}
	app := newApp(*dataDir)

	started := time.Now()
	if app.loadDisk() {
		app.rebuildLocked()
		snap := app.snap.Load()
		if snap != nil {
			log.Printf("snapshot loaded: %d titles in %s", len(snap.Entries), time.Since(started).Round(time.Millisecond))
		}
	}

	go app.crawlLoop()

	server := &http.Server{
		Addr:              *addr,
		Handler:           app.routes(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("listening on %s", *addr)
	log.Fatal(server.ListenAndServe())
}
