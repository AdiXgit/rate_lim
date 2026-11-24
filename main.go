package main

import (
	"log"
	"net/http"
	"time"
)

func main() {
	cfg, err := LoadSettings()
	if err != nil {
		log.Fatal(err)
	}
	store := NewStore(cfg.Capacity, cfg.RefillPerSec, cfg.IdleTTLSec, MonotonicClock())
	srv := &http.Server{
		Addr:              ":8000",
		Handler:           NewServer(cfg, store),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("listening on %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
