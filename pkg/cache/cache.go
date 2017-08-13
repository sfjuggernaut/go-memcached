package cache

import (
	"errors"
)

var (
	ErrCacheMiss = errors.New("Cache miss")
)

// A simple interface to allow for multiple caching strategies.
type Cache interface {
	Add(key, value string, flags uint32)
	Get(key string) (string, uint32, uint64, error)
	Delete(key string) error
}
