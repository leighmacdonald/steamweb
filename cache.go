package steamweb

import (
	"sync"
	"time"
)

var (
	cacheExpiry = 900
	cache       = &memoryCache{
		RWMutex: &sync.RWMutex{},
		values:  map[cacheKey]cacheValue{},
	}
)

type cacheKey int

const (
	ckApps cacheKey = iota
	ckAPIList
	ckStoreMetaData
	ckSchemaURL
	ckSchemaItems
	ckSchemaOverview
)

type cacheValue struct {
	Created time.Time
	MaxAge  time.Duration
	value   interface{}
}

func (v cacheValue) expired() bool {
	return time.Now().Sub(v.Created) > v.MaxAge
}

// memoryCache provides a very basic in-memory, generic cache with value timeouts
type memoryCache struct {
	*sync.RWMutex
	values map[cacheKey]cacheValue
}

// get retrieves any caches values from memory. If the value exists,
// but the cache has expired, the value will not be returned. You must not
// cast any value that was returned with a false status
func (c *memoryCache) get(key cacheKey) (interface{}, bool) {
	c.RLock()
	defer c.RUnlock()
	v, found := c.values[key]
	if !found {
		return nil, false
	}
	if v.expired() {
		return v, false
	}
	return v, true
}

// set will store the value in memory with a optional custom expiry time in seconds.
// If no expiry is provided, the default package level cacheExpiry will be used.
func (c *memoryCache) set(key cacheKey, value interface{}, customExpiry ...int) {
	c.Lock()
	defer c.Unlock()
	exp := cacheExpiry
	if len(customExpiry) > 0 {
		exp = customExpiry[0]
	}
	c.values[key] = cacheValue{
		Created: time.Now(),
		MaxAge:  time.Duration(exp) * time.Second,
		value:   value,
	}
}
