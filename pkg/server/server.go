package server

import (
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/sfjuggernaut/go-memcached/pkg/cache"
)

const (
	maxKeyLength = 250
)

// Server is the root structure of the memcached server.
// numWorkers is the number of goroutines to spin up to concurrently handle
// incoming connections.
// maxNumConnections is the buffer size of the channel to hold incoming
// connections. If the buffer gets full, then the server will block accepting
// new connections.
type Server struct {
	listener          net.Listener
	port              int
	numWorkers        int
	maxNumConnections int
	LRU               *cache.LRU
	quit              chan struct{}
	wg                sync.WaitGroup
}

func New(port int, capacity uint64, numWorkers, maxNumConnections int) *Server {
	return &Server{
		port:              port,
		numWorkers:        numWorkers,
		maxNumConnections: maxNumConnections,
		LRU:               cache.NewLRU(capacity),
		wg:                sync.WaitGroup{},
		quit:              make(chan struct{}),
	}
}

func (server *Server) connectionWorker(conns chan net.Conn) {
	defer server.wg.Done()

Loop:
	for {
		select {
		case conn := <-conns:
			server.handleConnection(conn)
		case <-server.quit:
			break Loop
		}
	}
}

func (s *Server) Start() {
	address := fmt.Sprintf(":%d", s.port)
	l, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal(err)
	}
	s.listener = l
	defer s.Stop()

	conns := make(chan net.Conn, s.maxNumConnections)

	// create workers to handle incoming connections
	for i := 0; i < s.numWorkers; i++ {
		s.wg.Add(1)
		go s.connectionWorker(conns)
	}

	for {
		// wait for a new connection
		conn, err := l.Accept()
		if err != nil {
			log.Print("Server: accept error:", err)
			continue
		}
		if conn == nil {
			log.Println("Server: received a nil conn, ignoring")
			continue
		}
		conns <- conn
	}
}

func (s *Server) Stop() {
	s.listener.Close()
	// wait for workers to cleanly shutdown
	close(s.quit)
	s.wg.Wait()
}
