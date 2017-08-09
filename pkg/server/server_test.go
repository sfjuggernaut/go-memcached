package server

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

func TestBasicTextProtocol(t *testing.T) {
	port := 22222
	srv := New(port, 100)
	go srv.Start()
	defer srv.Stop()

	address := fmt.Sprintf(":%d", port)
	client := memcache.New(address)

	waitForServerToStart()

	key := "k1"
	value := "wombat"

	// set
	if err := client.Set(&memcache.Item{Key: key, Value: []byte(value)}); err != nil {
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
	size := 10
	port := 33333
	srv := New(port, size)
	go srv.Start()
	defer srv.Stop()

	address := fmt.Sprintf(":%d", port)
	client := memcache.New(address)

	waitForServerToStart()

	// insert up to eviction size number of elements
	for i := 0; i < size; i++ {
		k := strconv.Itoa(i)
		if err := client.Set(&memcache.Item{Key: k, Value: []byte(k)}); err != nil {
			t.Errorf("Set of key (%s) got unexpected error: %s\n", k, err)
		}
	}

	// verify all the elements are found
	for i := 0; i < size; i++ {
		k := strconv.Itoa(i)
		if _, err := client.Get(k); err != nil {
			t.Errorf("Get of key (%s) got unexpected error: %s\n", k, err)
		}
	}

	// add one more element which should cause eviction
	k := strconv.Itoa(size)
	if err := client.Set(&memcache.Item{Key: k, Value: []byte(k)}); err != nil {
		t.Errorf("Set of key (%s) got unexpected error: %s\n", k, err)
	}

	// verify first item ever inserted is evicted and all others still exist
	k = strconv.Itoa(0)
	_, err := client.Get(k)
	if err == nil {
		t.Errorf("Get of key (%s) should have returned a memcache miss\n", k)
	}

}

// wait a little bit for the server to be able to receive connections
func waitForServerToStart() {
	time.Sleep(50 * time.Millisecond)
}
