package main

import (
	"flag"

	"github.com/sfjuggernaut/go-memcached/pkg/server"
)

var port = flag.Int("port", 11211, "port to run memcached server")
var size = flag.Int("size", 1024, "maximum number of entries to store")

func main() {
	flag.Parse()

	server := server.New(*port, *size)
	server.Start()
}
