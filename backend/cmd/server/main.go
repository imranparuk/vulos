package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"vulos/backend/internal/config"
)

func main() {
	env := flag.String("env", "local", "Environment: local, dev, main")
	flag.Parse()

	cfg := config.Load(*env)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok"}`)
	})

	addr := ":" + cfg.Port
	log.Printf("server listening on %s (env=%s)\n", addr, *env)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
