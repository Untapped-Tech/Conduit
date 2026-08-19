package cache

import (
	"container/list"
	"sync"
	"time"
)

const SentinelInvalidToken = "__none__"

type cacheElement struct {
	token     string
	role      string
	expiresAt time.Time
}

type TTLLRUCache struct {
	mutex       sync.RWMutex
	capacity    int
	ttl         time.Duration
	itemsMap    map[string]*list.Element
	elementList *list.List
}

func NewTTLLRUCache(capacity int, ttlSeconds int) *TTLLRUCache {
	if capacity <= 0 {
		capacity = 10000
	}
	ttlDuration := time.Duration(ttlSeconds) * time.Second
	if ttlDuration <= 0 {
		ttlDuration = 300 * time.Second
	}

	return &TTLLRUCache{
		capacity:    capacity,
		ttl:         ttlDuration,
		itemsMap:    make(map[string]*list.Element),
		elementList: list.New(),
	}
}

func (cache *TTLLRUCache) Get(tokenString string) (string, bool) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	element, exists := cache.itemsMap[tokenString]
	if !exists {
		return "", false
	}

	entry := element.Value.(*cacheElement)
	if time.Now().After(entry.expiresAt) {
		cache.elementList.Remove(element)
		delete(cache.itemsMap, tokenString)
		return "", false
	}

	cache.elementList.MoveToFront(element)
	return entry.role, true
}

func (cache *TTLLRUCache) Set(tokenString string, roleString string) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	if element, exists := cache.itemsMap[tokenString]; exists {
		cache.elementList.MoveToFront(element)
		entry := element.Value.(*cacheElement)
		entry.role = roleString
		entry.expiresAt = time.Now().Add(cache.ttl)
		return
	}

	if cache.elementList.Len() >= cache.capacity {
		oldestElement := cache.elementList.Back()
		if oldestElement != nil {
			cache.elementList.Remove(oldestElement)
			oldestEntry := oldestElement.Value.(*cacheElement)
			delete(cache.itemsMap, oldestEntry.token)
		}
	}

	newEntry := &cacheElement{
		token:     tokenString,
		role:      roleString,
		expiresAt: time.Now().Add(cache.ttl),
	}
	newElement := cache.elementList.PushFront(newEntry)
	cache.itemsMap[tokenString] = newElement
}
