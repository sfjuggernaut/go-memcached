package main

import (
	"flag"

	"github.com/sfjuggernaut/go-memcached/pkg/cache"
	"github.com/sfjuggernaut/go-memcached/pkg/server"
)

var port = flag.Int("port", 11211, "port to run memcached server")
var adminHttpPort = flag.Int("admin-http-port", 8989, "port to run admin HTTP server")
var capacity = flag.Uint64("capacity", 1024*1024*64, "maximum number of bytes to store (memory limit of server)")
var numWorkers = flag.Int("num-workers", 8, "number of workers to process incoming connections")
var maxNumConnections = flag.Int("max-num-connections", 1024, "maximum number of simultaneous connections")
var numBuckets = flag.Int("num-buckets", 16, "number of buckets in the hash table of the cache")

func main() {
	flag.Parse()

	cache := cache.NewLRU(*capacity, uint32(*numBuckets))
	server := server.New(*port, *adminHttpPort, *numWorkers, *maxNumConnections, cache)
	server.Start()
}
