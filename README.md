# go-memcached

A memcache server written in Go.

### Protocols currently supported
- TEXT

### Operations currently supported
- CAS
- DELETE
- GET
- GETS
- SET

## Testing

`
$ go test ./...
`

`
$ go tool vet pkg
$ go tool vet cmd
`

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