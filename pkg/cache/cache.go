package cache

import (
	"container/list"
	"errors"
	"log"
	"sync/atomic"
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
	casToken  uint64
}

// entry holds the information for an entry in the LRU's map.
type entry struct {
	key   string
	value string
	flags uint32
	cas   uint64
}

// NewLRU returns a new LRU object.
func NewLRU(size int) *LRU {
	return &LRU{size: size, evictList: list.New(), elements: make(map[string]*list.Element)}
}

// Add inserts or updates the element for the specified key.
func (lru *LRU) Add(key, value string, flags uint32) {
	if e, ok := lru.elements[key]; ok {
		lru.updateElement(e, value, flags)
	} else {
		lru.addElement(key, value, flags)
	}
	lru.checkSize()
}

// Get retrieves the value and cas token stored in the element
// for the specified key.
// Returns error if element is not found.
func (lru *LRU) Get(key string) (string, uint32, uint64, error) {
	e, ok := lru.elements[key]
	if !ok {
		return "", 0, 0, ErrCacheMiss
	}
	lru.refreshElement(e)

	return e.Value.(*entry).value, e.Value.(*entry).flags, e.Value.(*entry).cas, nil
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

// atomically increment and return unique cas token
func (lru *LRU) getNewCasToken() uint64 {
	return atomic.AddUint64(&lru.casToken, 1)
}

// add element to cache and update LRU for this element
func (lru *LRU) addElement(key, value string, flags uint32) {
	e := lru.evictList.PushFront(&entry{key: key, value: value, flags: flags, cas: lru.getNewCasToken()})
	lru.elements[key] = e
}

// update element to cache and update LRU for this element
func (lru *LRU) updateElement(e *list.Element, value string, flags uint32) {
	e.Value.(*entry).value = value
	e.Value.(*entry).flags = flags
	e.Value.(*entry).cas = lru.getNewCasToken()
	lru.evictList.MoveToFront(e)
}

// update LRU for this element
func (lru *LRU) refreshElement(e *list.Element) {
	lru.evictList.MoveToFront(e)
}

// remove element
func (lru *LRU) deleteElement(e *list.Element) {
	delete(lru.elements, e.Value.(*entry).key)
	lru.evictList.Remove(e)
}

// remove last element in evictList if we have more then `size` elements cached
func (lru *LRU) checkSize() {
	if lru.evictList.Len() > lru.size {
		e := lru.evictList.Back()
		lru.deleteElement(e)
	}
}
