package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"task278-broadcastslot/internal/httpapi"
	"task278-broadcastslot/internal/service"
	"task278-broadcastslot/internal/smoke"
	"task278-broadcastslot/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "./broadcastslot.db", "SQLite database path")
	smokeTest := flag.Bool("smoke-test", false, "run end-to-end smoke test")
	flag.Parse()

	if *smokeTest {
		path := *dbPath
		if !flagPassed("db") {
			dir, err := os.MkdirTemp("", "broadcastslot-smoke-*")
			if err != nil {
				log.Fatal(err)
			}
			path = filepath.Join(dir, "smoke.db")
		}
		if err := smoke.Run(path); err != nil {
			log.Fatalf("smoke test failed: %v", err)
		}
		return
	}

	st, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer st.Close()

	svc := service.New(st)
	srv := httpapi.NewServer(svc)
	log.Printf("broadcastslot listening on %s db=%s", *addr, *dbPath)
	if err := http.ListenAndServe(*addr, srv.Handler()); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func flagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
