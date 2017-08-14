package cache

import (
	"container/list"
	"hash/fnv"
	"log"
	"sync"
	"sync/atomic"
)

// This implements a very straight forward LRU using buckets of maps and doubly linked lists.
// The `capacity` parameter is the approximate maximum number of bytes that can be
// stored until eviction occurs.
//
// Rather than a single global evict list for all buckets, each bucket has its own evict list
// to allow us to process more requests concurrently. We trade-off not guaranteeing the truest
// least recently used item being evicted for better performance.
//
// To keep track of the number of objects instead of bytes, have "entry.size()" always return 1.
//
// Doesn't handle pre-allocation of memory.
type LRU struct {
	// approximate maximum number of bytes to be stored (never changes)
	capacity uint64

	// number of buckets to hash across
	numBuckets uint32

	// table and evict list for entries hashed into each bucket
	buckets []*Bucket

	// table of entries stored (k: key of entry)
	// elements map[string]*list.Element

	// unique token counter for inserts and updates
	casToken uint64
}

// Bucket implements a simple hash and LRU using a doubly linked list.
// The `capacity` parameter is the approximate maximum number of bytes that can be
// stored until eviction occurs.
type Bucket struct {
	// approximate maximum number of bytes to be stored (never changes)
	capacity uint64

	// current number of bytes stored
	size uint64

	// table of entries stored (k: key of entry)
	elements map[string]*list.Element

	// doubly linked list for entries to be evicted
	evictList *list.List

	// protects access to:
	// - elements
	// - evicList
	// - size
	sync.RWMutex
}

// entry holds the information for an entry in the Bucket's map.
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
func NewLRU(capacity uint64, numBuckets uint32) *LRU {
	buckets := make([]*Bucket, numBuckets)
	for i := uint32(0); i < numBuckets; i++ {
		b := &Bucket{
			capacity:  capacity / uint64(numBuckets),
			elements:  make(map[string]*list.Element),
			evictList: list.New(),
		}
		buckets[i] = b
	}
	return &LRU{capacity: capacity, numBuckets: numBuckets, buckets: buckets}
}

// Add inserts or updates the element for the specified key.
func (lru *LRU) Add(key, value string, flags uint32) {
	bucket := lru.buckets[lru.hash(key)%lru.numBuckets]
	newCas := lru.getNewCasToken()

	bucket.Lock()
	defer bucket.Unlock()

	if e, ok := bucket.elements[key]; ok {
		bucket.updateElement(e, value, flags, newCas)
	} else {
		bucket.addElement(key, value, flags, newCas)
	}
	bucket.checkCapacity()
}

// Get retrieves the value and cas token stored in the element
// for the specified key.
// Returns error if element is not found.
func (lru *LRU) Get(key string) (string, uint32, uint64, error) {
	bucket := lru.buckets[lru.hash(key)%lru.numBuckets]

	bucket.RLock()
	defer bucket.RUnlock()

	e, ok := bucket.elements[key]
	if !ok {
		return "", 0, 0, ErrCacheMiss
	}
	bucket.refreshElement(e)

	return e.Value.(*entry).value, e.Value.(*entry).flags, e.Value.(*entry).cas, nil
}

// Delete removes the element for the specified key.
// Returns error if element is not found.
func (lru *LRU) Delete(key string) error {
	bucket := lru.buckets[lru.hash(key)%lru.numBuckets]

	bucket.Lock()
	defer bucket.Unlock()

	e, ok := bucket.elements[key]
	if !ok {
		return ErrCacheMiss
	}
	bucket.deleteElement(e)

	return nil
}

// hash returns the hash of the specified key
func (lru *LRU) hash(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32()
}

// atomically increment and return unique cas token
func (lru *LRU) getNewCasToken() uint64 {
	return atomic.AddUint64(&lru.casToken, 1)
}

// PrintEvictList is an aid for debugging.
func (bucket *Bucket) PrintEvictList() {
	bucket.Lock()
	defer bucket.Unlock()

	s := "Bucket evict list: front->"
	for e := bucket.evictList.Front(); e != nil; e = e.Next() {
		entryValue := e.Value.(*entry)
		s += entryValue.key + "<->"
	}
	s += "back"
	log.Println(s)
}

// add element to cache and update evict list for this element
func (bucket *Bucket) addElement(key, value string, flags uint32, cas uint64) {
	e := bucket.evictList.PushFront(&entry{key: key, value: value, flags: flags, cas: cas})
	bucket.elements[key] = e
	bucket.size += e.Value.(*entry).size()
}

// update element in cache and update evict list for this element
func (bucket *Bucket) updateElement(e *list.Element, value string, flags uint32, cas uint64) {
	oldSize := e.Value.(*entry).size()
	e.Value.(*entry).value = value
	e.Value.(*entry).flags = flags
	e.Value.(*entry).cas = cas
	bucket.evictList.MoveToFront(e)
	bucket.size += e.Value.(*entry).size() - oldSize
}

// update evict list for this element
func (bucket *Bucket) refreshElement(e *list.Element) {
	bucket.evictList.MoveToFront(e)
}

// remove element from cache and evict list
func (bucket *Bucket) deleteElement(e *list.Element) {
	delete(bucket.elements, e.Value.(*entry).key)
	bucket.evictList.Remove(e)
	bucket.size -= e.Value.(*entry).size()
}

// remove last element in evict list if we have more than 'capacity' bytes
func (bucket *Bucket) checkCapacity() {
	for bucket.size > bucket.capacity {
		e := bucket.evictList.Back()
		if e == nil {
			log.Println("want to evict but found nothing on the evict list, this should rarely happen")
			break
		}
		bucket.deleteElement(e)
	}
}
