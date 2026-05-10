package main

import (
	"errors"
	"log/slog"
	"net/http"

	"dns-manager/server/core"
)

const path = "dns/resolv.conf"

func main() {
	log := slog.Default()
	m := core.NewManager(path)
	s := core.NewServer(log, m)

	mux := http.NewServeMux()
	mux.Handle("POST /add", s.HandleAdd())
	mux.Handle("DELETE /add", s.HandleRemove())
	mux.Handle("GET /list", s.HandleList())

	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error("server closed unexpectedly", "error", err)
			return
		}
	}
}
