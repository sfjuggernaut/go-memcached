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
