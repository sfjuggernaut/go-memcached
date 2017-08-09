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
// The `size` parameter is the maximum number of elements that can be stored until
// eviction occurs. It is not the total number of bytes stored.
//
// Not thread safe.
// Doesn't handle pre-allocation of memory.
type LRU struct {
	size      int
	elements  map[string]*list.Element
	evictList *list.List
}

// entry holds the information for an entry in the LRU's map.
type entry struct {
	key   string
	value string
}

func NewLRU(size int) *LRU {
	return &LRU{size: size, evictList: list.New(), elements: make(map[string]*list.Element)}
}

// Add inserts or updates the element for the specified key.
func (lru *LRU) Add(key, value string) {
	if e, ok := lru.elements[key]; ok {
		lru.updateElement(e, value)
	} else {
		lru.addElement(key, value)
	}
	lru.checkSize()
}

// Get retrieves the value stored in the element for the specified key.
// Returns error if element is not found.
func (lru *LRU) Get(key string) (string, error) {
	e, ok := lru.elements[key]
	if !ok {
		return "", ErrCacheMiss
	}
	lru.refreshElement(e)

	return e.Value.(*entry).value, nil
}

// Delete removes the element for the specified key.
// Returns error if element is not found.
func (lru *LRU) Delete(key string) error {
	e, ok := lru.elements[key]
	if !ok {
		return ErrCacheMiss
	}
	lru.deleteElement(e)

	return nil
}

// PrintEvictList is an aid for debugging.
func (lru *LRU) PrintEvictList() {
	log.Println("LRU: evict list")
	s := "LRU evict list: front->"
	for e := lru.evictList.Front(); e != nil; e = e.Next() {
		entryValue := e.Value.(*entry)
		s += entryValue.key + "<->"
	}
	s += "back"
	log.Println(s)
}

func (lru *LRU) addElement(key, value string) {
	e := lru.evictList.PushFront(&entry{key: key, value: value})
	lru.elements[key] = e
}

func (lru *LRU) updateElement(e *list.Element, value string) {
	e.Value.(*entry).value = value
	lru.evictList.MoveToFront(e)
}

func (lru *LRU) refreshElement(e *list.Element) {
	lru.evictList.MoveToFront(e)
}

func (lru *LRU) deleteElement(e *list.Element) {
	delete(lru.elements, e.Value.(*entry).key)
	lru.evictList.Remove(e)
}

// Remove last element in evictList if we have more then `size` elements cached.
func (lru *LRU) checkSize() {
	if lru.evictList.Len() > lru.size {
		e := lru.evictList.Back()
		lru.deleteElement(e)
	}
}
