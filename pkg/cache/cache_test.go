package cache

import (
	"sync"
	"testing"
)

// An example cache that adheres to the Cache interface.
// Only caches the last entry set.
type LastEntryCache struct {
	key   string
	value string
	flags uint32
	cas   uint64

	sync.RWMutex
}

func NewLEC() *LastEntryCache {
	return &LastEntryCache{}
}

func (l *LastEntryCache) Add(key, value string, flags uint32) {
	l.Lock()
	defer l.Unlock()

	l.key = key
	l.value = value
	l.flags = flags
	l.cas += 1
}
func (l *LastEntryCache) Get(key string) (string, uint32, uint64, error) {
	l.RLock()
	defer l.RUnlock()

	if key != l.key {
		return "", 0, 0, ErrCacheMiss
	}
	return l.value, l.flags, l.cas, nil
}

func (l *LastEntryCache) Delete(key string) error {
	l.Lock()
	defer l.Unlock()

	if key != l.key {
		return ErrCacheMiss
	}
	l.key = ""
	return nil
}

func TestLCEAdd(t *testing.T) {
	cache := NewLEC()

	//
	// Verify Add of first entry
	//

	// add first entry
	key1 := "k1"
	value1 := "wombat"
	cache.Add(key1, value1, 0)

	// verify its found
	data, _, _, err := cache.Get(key1)
	if err != nil {
		t.Errorf("GET for key (%s) received unexpected err: %s\n", key1, err)
	}
	if data != value1 {
		t.Errorf("GET for key (%s) expected value (%s) but received (%s) instead\n", key1, value1, data)
	}

	//
	// Add second entry
	//

	// first verify new entry isn't found
	key2 := "k2"
	value2 := "zoo"
	if _, _, _, err := cache.Get(key2); err != ErrCacheMiss {
		t.Errorf("GET for key (%s) expected (%s) but received (%s)\n", key2, ErrCacheMiss, err)
	}

	// add second entry
	cache.Add(key2, value2, 0)

	// verify key2 is found with correct data
	data, _, _, err = cache.Get(key2)
	if err != nil {
		t.Errorf("GET for key (%s) received unexpected err: %s\n", key2, err)
	}
	if data != value2 {
		t.Errorf("GET for key (%s) expected value (%s) but received (%s) instead\n", key2, value2, data)
	}

	// verify first entry is no longer found
	if _, _, _, err := cache.Get(key1); err != ErrCacheMiss {
		t.Errorf("GET for key (%s) expected (%s) but received (%s)\n", key1, ErrCacheMiss, err)
	}

	//
	// Delete second entry
	//

	// verify delete of second entry
	if err := cache.Delete(key2); err != nil {
		t.Errorf("DELETE for key (%s) received unexpected err: %s\n", key2, err)
	}

	// verify second entry is no longer found
	if _, _, _, err := cache.Get(key2); err != ErrCacheMiss {
		t.Errorf("GET for key (%s) expected (%s) but received (%s)\n", key2, ErrCacheMiss, err)
	}
}
