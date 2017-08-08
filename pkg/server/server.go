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
	port int
	LRU  *cache.LRU
}

func New(port, size int) *Server {
	lru := cache.NewLRU(size)
	return &Server{port: port, LRU: lru}
}

func (server *Server) Start() {
	address := fmt.Sprintf(":%d", server.port)
	l, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	for {
		// wait for a new connection
		conn, err := l.Accept()
		if err != nil {
			log.Print(err)
			break
		}

		// handle the connection in a new goroutine
		go server.handleConnection(conn)
	}
}
