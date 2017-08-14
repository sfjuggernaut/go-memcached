package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"time"
)

const (
	defaultShutdownDelay = 2 * time.Second
)

// This admin HTTP port allows one to query the memcached
// server to retrieve stats via HTTP (instead of the memcache protocol).

func (s *Server) adminHttpServerStart(port int) {
	mux := http.NewServeMux()
	mux.HandleFunc("/stats", s.getStatsHandler)
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)

	address := fmt.Sprintf(":%d", port)
	httpServer := &http.Server{Addr: address, Handler: mux}
	s.adminHttpServer = httpServer
	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			log.Printf("listen received err: %s\n", err)
		}
	}()
}

func (s *Server) adminHttpServerStop() {
	ctx, _ := context.WithTimeout(context.Background(), defaultShutdownDelay)
	s.adminHttpServer.Shutdown(ctx)
}

func (s *Server) getStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats := s.getStats()
	data, err := json.Marshal(stats)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(data)
}
