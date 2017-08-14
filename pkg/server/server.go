package server

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/sfjuggernaut/go-memcached/pkg/cache"
)

const (
	maxKeyLength = 250
)

// Server is the root structure of the memcached server.
// 'numWorkers' is the number of goroutines to spin up to concurrently handle
// incoming connections.
// 'maxNumConnections' is the buffer size of the channel to hold incoming
// connections. If the buffer gets full, then the server will block accepting
// new connections.
type Server struct {
	listener          net.Listener
	port              int
	adminHttpPort     int
	numWorkers        int
	maxNumConnections int
	Cache             cache.Cache
	adminHttpServer   *http.Server
	startTime         time.Time
	quit              chan struct{}
	wg                sync.WaitGroup
}

// New returns a new Server.
func New(port, adminHttpPort, numWorkers, maxNumConnections int, cache cache.Cache) *Server {
	return &Server{
		port:              port,
		adminHttpPort:     adminHttpPort,
		numWorkers:        numWorkers,
		maxNumConnections: maxNumConnections,
		Cache:             cache,
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

// Start function starts listing for incoming TCP requests
// and also starts up an admin HTTP server.
func (s *Server) Start() {
	s.startTime = time.Now().UTC()
	s.adminHttpServerStart(s.adminHttpPort)

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
			// log.Print("Server: accept error:", err)
			continue
		}
		if conn == nil {
			log.Println("Server: received a nil conn, ignoring")
			continue
		}
		conns <- conn
	}
}

// Stop cleanly shutdowns the Server (and its dependencies).
func (s *Server) Stop() {
	s.listener.Close()
	// wait for workers to cleanly shutdown
	close(s.quit)
	// shutdown admin http server
	s.adminHttpServerStop()
	s.wg.Wait()
}
