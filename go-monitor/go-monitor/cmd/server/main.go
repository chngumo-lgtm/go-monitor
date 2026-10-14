package main

import (
	"log"
	"net/http"
	"os"

	"github.com/chngumo-lgtm/go-monitor/internal/monitor"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	srv := monitor.NewServer(monitor.NewStore(), monitor.HTTPChecker)
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, srv.Routes()))
}
