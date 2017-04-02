// package cache implements a really primitive cache that associates expiring
// values with string keys.  This cache never clears itself out.
package mitm

import (
	"sync"
	"time"
)

// Cache is a cache for binary data
type Cache struct {
	entries map[string]*entry
	mutex   sync.RWMutex
}

// entry is an entry in a Cache
type entry struct {
	data       interface{}
	expiration time.Time
}

// NewCache creates a new Cache
func NewCache() *Cache {
	return &Cache{entries: make(map[string]*entry)}
}

// Get returns the currently cached value for the given key, as long as it
// hasn't expired.  If the key was never set, or has expired, found will be
// false.
func (cache *Cache) Get(key string) (val interface{}, found bool) {
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()
	entry := cache.entries[key]
	if entry == nil {
		return nil, false
	} else if entry.expiration.Before(time.Now()) {
		return nil, false
	} else {
		return entry.data, true
	}
}

// Set sets a value in the cache with an expiration of now + ttl.
func (cache *Cache) Set(key string, data interface{}, ttl time.Duration) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	cache.entries[key] = &entry{data, time.Now().Add(ttl)}
}
