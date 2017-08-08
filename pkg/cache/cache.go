package cache

import (
	"container/list"
	"errors"
	"log"
)

var (
	ErrCacheMiss = errors.New("Cache miss")
)

// This implements a very straight forward LRU using a map and a doubly linked list.
//
// Not thread safe.
// Doesn't handle pre-allocation of memory.
type LRU struct {
	size      int
	elements  map[string]*list.Element
	evictList *list.List
}

func NewLRU(size int) *LRU {
	return &LRU{size: size, evictList: list.New(), elements: make(map[string]*list.Element)}
}

// Add inserts or updates the element for the specified key.
func (lru *LRU) Add(key, value string) {
	if e, ok := lru.elements[key]; ok {
		e.Value = value
		lru.evictList.MoveToFront(e)
	} else {
		e := lru.evictList.PushFront(value)
		lru.elements[key] = e
	}

	if lru.evictList.Len() > lru.size {
		lru.evictList.Remove(lru.evictList.Back())
	}
}

// Get retrieves the value of the element for the specified key.
// Returns error if element is not found.
func (lru *LRU) Get(key string) (string, error) {
	e, ok := lru.elements[key]
	if !ok {
		return "", ErrCacheMiss
	}
	lru.evictList.MoveToFront(e)

	return e.Value.(string), nil
}

// Delete removes the element for the specified key.
// Returns error if element is not found.
func (lru *LRU) Delete(key string) error {
	e, ok := lru.elements[key]
	if !ok {
		log.Printf("LRU: delete didn't find (%s)\n", key)
		return ErrCacheMiss
	}
	delete(lru.elements, key)
	lru.evictList.Remove(e)
	return nil
}

// PrintEvictList is useful for debugging.
func (lru *LRU) PrintEvictList() {
	log.Println("LRU: evict list")
	s := "LRU evict list: front->"
	for e := lru.evictList.Front(); e != nil; e = e.Next() {
		s += e.Value.(string) + "<->"
	}
	s += "back"
	log.Println(s)
}
