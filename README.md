# go-memcached

A memcache server written in Go.

### Interfaces currently supported
- TCP

### Protocols currently supported
- TEXT

### Operations currently supported
- CAS
- DELETE
- GET
- GETS
- SET

## Documentation

Use [godoc](http://godoc.org/golang.org/x/tools/cmd/godoc):

`
$ godoc -http=:8080
`

Then navigate to:

```
http://localhost:8080/pkg/github.com/sfjuggernaut/go-memcached/
http://localhost:8080/pkg/github.com/sfjuggernaut/go-memcached/pkg/cache/
http://localhost:8080/pkg/github.com/sfjuggernaut/go-memcached/pkg/server
```

## Testing

```
$ go test ./...
```

```
$ go tool vet pkg
$ go tool vet cmd
```

```
$ go test -race pkg/server/*.go
$ go test -race pkg/cache/*.go
```

## Update dependencies via [godep](godephttps://github.com/tools/godep)

`
$ godep save ./...
`

## Installation and Running

Tested and built using go version `1.8.3` on OSX `10.12.6` and Debian `4.9.30`.

To run manually:

```
$ which go
/usr/local/bin/go
$ echo $GOPATH
/Users/coolstars/Code/go
$ pwd
/Users/coolstars/Code/go/src/github.com/sfjuggernaut/go-memcached
$ go run cmd/go-memcached/main.go
```

To build and install:

```
$ which go
/usr/local/bin/go
$ echo $GOPATH
/Users/coolstars/Code/go
$ pwd
/Users/coolstars/Code/go/src/github.com/sfjuggernaut/go-memcached
$ go build github.com/sfjuggernaut/go-memcached/cmd/go-memcached/
$ go install github.com/sfjuggernaut/go-memcached/cmd/go-memcached/
```

To run newly created binary:

```
$ ~/Code/go/bin/go-memcached
```

## Profiling

Endpoints for [profiling](https://blog.golang.org/profiling-go-programs) via [pprof](https://golang.org/pkg/net/http/pprof/) are exposed via the admin HTTP interface at `/debug/profile/`.

Examples:

```
$ go tool pprof http://localhost:8989/debug/pprof/heap
$ go tool pprof http://localhost:8989/debug/pprof/heap
```
