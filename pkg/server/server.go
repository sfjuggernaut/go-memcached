package server

import (
	"fmt"
	"log"
	"net"

	"github.com/sfjuggernaut/go-memcached/pkg/cache"
)

const (
	maxKeyLength = 250
)

type Server struct {
	listener net.Listener
	port     int
	LRU      *cache.LRU
}

func New(port, size int) *Server {
	lru := cache.NewLRU(size)
	return &Server{port: port, LRU: lru}
}

func (s *Server) Start() {
	address := fmt.Sprintf(":%d", s.port)
	l, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal(err)
	}
	s.listener = l
	defer s.Stop()

	for {
		// wait for a new connection
		conn, err := l.Accept()
		if err != nil {
			log.Print(err)
			break
		}

		// handle the connection in a new goroutine
		go s.handleConnection(conn)
	}
}

func (s *Server) Stop() {
	s.listener.Close()
}
