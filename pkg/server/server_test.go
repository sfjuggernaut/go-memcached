package server

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/sfjuggernaut/go-memcached/pkg/cache"
)

func TestBasicTextProtocol(t *testing.T) {
	port := 22222
	cache := cache.NewLRU(1024 * 1024)
	srv := New(port, 8000, 8, 1024, cache)
	go srv.Start()
	defer srv.Stop()

	address := fmt.Sprintf(":%d", port)
	client := memcache.New(address)

	waitForServerToStart()

	key := "k1"
	value := "wombat"
	flags := uint32(13)

	// set
	item := &memcache.Item{Key: key, Value: []byte(value), Flags: flags}
	if err := client.Set(item); err != nil {
		t.Errorf("Set of key (%s) got unexpected error: %s\n", key, err)
	}

	// get
	it, err := client.Get(key)
	if err != nil {
		t.Errorf("Get of key (%s) got unexpected error: %s\n", key, err)
	}
	if it == nil {
		t.Fatalf("Get of key (%s) got unexpected nil item returned\n", key)
	}
	if string(it.Value) != value {
		t.Errorf("Get of key (%s) got unexpected value: %s\n", key, it.Value)
	}
	if it.Flags != flags {
		t.Errorf("Get of key (%s) expected flags to be (%d) but received (%d)\n", key, flags, it.Flags)
	}

	// delete
	if err := client.Delete(key); err != nil {
		t.Errorf("Delete of key (%s) got unexpected error: %s\n", key, err)
	}

	// get after delete
	it, err = client.Get(key)
	if err == nil {
		t.Errorf("Get of key (%s) expected cache miss, but go no error\n", key)
	}
}

func TestEviction(t *testing.T) {
	numEntries := 5
	// set capacity to one more than 'numEntries' entries worth of bytes
	capacity := uint64(numEntries*10 + 1)
	cache := cache.NewLRU(capacity)

	port := 33333
	srv := New(port, 8001, 8, 1024, cache)
	go srv.Start()
	defer srv.Stop()

	address := fmt.Sprintf(":%d", port)
	client := memcache.New(address)

	waitForServerToStart()

	// Insert 'numEntries' number of 10 byte entries.
	// The total should be just under capacity.
	// Note: key is 1 byte, value is 9 bytes.
	// Note: assumes 'numEntries' is less than 10 to produce a single byte key.
	value := "123456789"
	for i := 0; i < numEntries; i++ {
		k := strconv.Itoa(i)
		if err := client.Set(&memcache.Item{Key: k, Value: []byte(value)}); err != nil {
			t.Errorf("Set of key (%s) got unexpected error: %s\n", k, err)
		}
	}

	// verify all the elements are found
	for i := 0; i < numEntries; i++ {
		k := strconv.Itoa(i)
		if _, err := client.Get(k); err != nil {
			t.Errorf("Get of key (%s) got unexpected error: %s\n", k, err)
		}
	}

	// add one more element which should cause eviction
	k := strconv.Itoa(numEntries)
	if err := client.Set(&memcache.Item{Key: k, Value: []byte(value)}); err != nil {
		t.Errorf("Set of key (%s) got unexpected error: %s\n", k, err)
	}

	// verify first item ever inserted is evicted and all others still exist
	k = strconv.Itoa(0)
	_, err := client.Get(k)
	if err == nil {
		t.Errorf("Get of key (%s) should have returned a memcache miss\n", k)
	}
}

func TestKeys(t *testing.T) {
	cache := cache.NewLRU(1024 * 1024)
	port := 44444
	srv := New(port, 8002, 8, 1024, cache)
	go srv.Start()
	defer srv.Stop()

	address := fmt.Sprintf(":%d", port)
	client := memcache.New(address)

	waitForServerToStart()

	// ensure key of maxKeyLength is valid
	bytes := make([]byte, maxKeyLength)
	for i := 0; i < maxKeyLength; i++ {
		bytes[i] = 'a'
	}

	if err := client.Set(&memcache.Item{Key: string(bytes), Value: []byte("value")}); err != nil {
		t.Errorf("Set of key with len (%d) should not have received error: %s\n", len(string(bytes)), err)
	}

	// ensure key one char longer than maxKeyLength is invalid
	bytes = make([]byte, maxKeyLength+1)
	for i := 0; i < maxKeyLength+1; i++ {
		bytes[i] = 'a'
	}

	if err := client.Set(&memcache.Item{Key: string(bytes), Value: []byte("value")}); err == nil {
		t.Errorf("Set of key with len (%d) should have received error\n", len(string(bytes)))
	}
}

func TestCAS(t *testing.T) {
	cache := cache.NewLRU(1024 * 1024)
	port := 55555
	srv := New(port, 8003, 8, 1024, cache)
	go srv.Start()
	defer srv.Stop()

	address := fmt.Sprintf(":%d", port)
	client := memcache.New(address)

	waitForServerToStart()

	key := "k1"
	value1 := "wombat"
	value2 := "zoo"

	//
	// Test successful case for CAS
	//

	// set original value
	item := &memcache.Item{Key: key, Value: []byte(value1)}
	err := client.Set(item)
	if err != nil {
		t.Errorf("Set of key (%s) received unexpected error: %s\n", key, err)
	}

	// retrieve cas value (note: memcache pkg's Get function actually calls GETS under the covers)
	item, err = client.Get(key)
	if err != nil {
		t.Errorf("Get of key (%s) received unexpected error: %s\n", key, err)
	}

	// set new value via CAS
	item.Value = []byte(value2)
	if err := client.CompareAndSwap(item); err != nil {
		t.Errorf("CAS of key (%s) received unexpected error: %s\n", key, err)
	}

	// ensure new value is stored
	item, err = client.Get(key)
	if err != nil {
		t.Errorf("Get of key (%s) received unexpected derror: %s\n", key, err)
	}
	if string(item.Value) != value2 {
		t.Errorf("Expected value of (%s) for key (%s), but received (%s)\n", string(item.Value), key, value2)
	}

	//
	// Verify case where key does not exist
	//

	// first get updated cas value (stored in the returned 'item')
	item, err = client.Get(key)
	if err != nil {
		t.Errorf("Get of key (%s) received unexpected derror: %s\n", key, err)
	}

	// set key to non-existent value
	item.Key = "this-key-is-not-stored-on-the-server"

	// verify we get the expected error (NOT_FOUND, which is ErrCacheMiss in memcache pkg)
	err = client.CompareAndSwap(item)
	if err == nil {
		t.Errorf("CAS of key (%s) should have received NOT_FOUND error but got no error\n", key)
	}
	if err != memcache.ErrCacheMiss {
		t.Errorf("Expected error was (%s) but received (%s)\n", memcache.ErrCacheMiss, err)
	}

	//
	// Verify case where a second client updates the value in between
	// the first client's GETS and CAS.
	//

	// first get updated cas value (stored in the returned 'item')
	item, err = client.Get(key)
	if err != nil {
		t.Errorf("Get of key (%s) received unexpected derror: %s\n", key, err)
	}

	// have a second client update 'key'
	client2 := memcache.New(address)
	item2 := &memcache.Item{Key: key, Value: []byte("client2")}
	if err := client2.Set(item2); err != nil {
		t.Errorf("Set of key (%s) received unexpected error: %s\n", key, err)
	}

	// have first client attempt to CAS with now outdated cas value
	err = client.CompareAndSwap(item)
	if err == nil {
		t.Errorf("CAS of key (%s) should have received EXISTS error but got no error\n", key)
	}
	if err != memcache.ErrCASConflict {
		t.Errorf("Expected error was (%s) but received (%s)\n", memcache.ErrCASConflict, err)
	}

}

// wait a little bit for the server to be able to receive connections
func waitForServerToStart() {
	time.Sleep(50 * time.Millisecond)
}
