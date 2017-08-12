package main

import (
	"flag"

	"github.com/sfjuggernaut/go-memcached/pkg/server"
)

var port = flag.Int("port", 11211, "port to run memcached server")
var capacity = flag.Uint64("capacity", 1024*1024*64, "maximum number of bytes to store (memory limit of server)")
var numWorkers = flag.Int("num-workers", 8, "number of workers to process incoming connections")
var maxNumConnections = flag.Int("max-num-connections", 1024, "maximum number of simultaneous connections")

func main() {
	flag.Parse()

	server := server.New(*port, *capacity, *numWorkers, *maxNumConnections)
	server.Start()
}
