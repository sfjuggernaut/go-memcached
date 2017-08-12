package cache

import (
	"container/list"
	"errors"
	"log"
	"sync"
	"sync/atomic"
)

var (
	ErrCacheMiss = errors.New("Cache miss")
)

// This implements a very straight forward LRU using a map and a doubly linked list.
// The `capacity` parameter is the approximate maximum number of bytes that can be
// stored until eviction occurs.
//
// To keep track of the number of objects instead of bytes, have "entry.size()" always return 1.
//
// Doesn't handle pre-allocation of memory.
type LRU struct {
	// approximate maximum number of bytes to be stored (never changes)
	capacity uint64

	// current number of bytes stored
	size uint64

	// table of entries stored (k: key of entry)
	elements map[string]*list.Element

	// doubly linked list for entries to be evicted
	evictList *list.List

	// unique token counter for inserts and updates
	casToken uint64

	// protects access to:
	// - elements
	// - evictList
	// - size
	sync.RWMutex
}

// entry holds the information for an entry in the LRU's map.
type entry struct {
	key   string
	value string
	flags uint32
	cas   uint64
}

// size returns an approximate count of bytes for an entry
func (e *entry) size() uint64 {
	return uint64(len(e.key) + len(e.value))
}

// NewLRU returns a new LRU object.
func NewLRU(capacity uint64) *LRU {
	return &LRU{capacity: capacity, evictList: list.New(), elements: make(map[string]*list.Element)}
}

// Add inserts or updates the element for the specified key.
func (lru *LRU) Add(key, value string, flags uint32) {
	lru.Lock()
	defer lru.Unlock()

	if e, ok := lru.elements[key]; ok {
		lru.updateElement(e, value, flags)
	} else {
		lru.addElement(key, value, flags)
	}
	lru.checkCapacity()
}

// Get retrieves the value and cas token stored in the element
// for the specified key.
// Returns error if element is not found.
func (lru *LRU) Get(key string) (string, uint32, uint64, error) {
	lru.RLock()
	defer lru.RUnlock()

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
	lru.Lock()
	defer lru.Unlock()

	e, ok := lru.elements[key]
	if !ok {
		return ErrCacheMiss
	}
	lru.deleteElement(e)

	return nil
}

// PrintEvictList is an aid for debugging.
func (lru *LRU) PrintEvictList() {
	lru.Lock()
	defer lru.Unlock()

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
	lru.size += e.Value.(*entry).size()
}

// update element to cache and update LRU for this element
func (lru *LRU) updateElement(e *list.Element, value string, flags uint32) {
	oldSize := e.Value.(*entry).size()
	e.Value.(*entry).value = value
	e.Value.(*entry).flags = flags
	e.Value.(*entry).cas = lru.getNewCasToken()
	lru.evictList.MoveToFront(e)
	lru.size += e.Value.(*entry).size() - oldSize
}

// update LRU for this element
func (lru *LRU) refreshElement(e *list.Element) {
	lru.evictList.MoveToFront(e)
}

// remove element
func (lru *LRU) deleteElement(e *list.Element) {
	delete(lru.elements, e.Value.(*entry).key)
	lru.evictList.Remove(e)
	lru.size -= e.Value.(*entry).size()
}

// remove last element in evictList if we have more than 'capacity' bytes
func (lru *LRU) checkCapacity() {
	for lru.size > lru.capacity {
		e := lru.evictList.Back()
		if e == nil {
			log.Println("want to evict but found nothing on the evict list, this should never happen")
			break
		}
		lru.deleteElement(e)
	}
}
