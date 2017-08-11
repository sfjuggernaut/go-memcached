package main

import (
	"flag"

	"github.com/sfjuggernaut/go-memcached/pkg/server"
)

var port = flag.Int("port", 11211, "port to run memcached server")
var size = flag.Int("size", 1024, "maximum number of entries to store")
var numWorkers = flag.Int("num-workers", 8, "number of workers to process incoming connections")
var maxNumConnections = flag.Int("max-num-connections", 1024, "maximum number of simultaneous connections")

func main() {
	flag.Parse()

	server := server.New(*port, *size, *numWorkers, *maxNumConnections)
	server.Start()
}
