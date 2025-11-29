package storage

import (
	"errors"
	"log/slog"
	"strings"
	"sync"
	"container/list"

	"github.com/behummble/29-11-2025/internal/config"
)

type Storage struct {
	links map[int][]string
	cache  *cache
	log *slog.Logger
	id int
}

type cache struct {
	cache map[string]string
	mutex *sync.RWMutex
}

func NewStorage(cfg config.StorageConfig, log *slog.Logger) *Storage {
	return &Storage{
		links: make(map[int][]string, cfg.LinksSize),
		cache: newCache(cfg.CacheSize),
		log: log,
	}
}

func(s *Storage) WriteLinksPackage(links []string, newLinks map[string]string) (int, error) {
	for i := 0; i < len(links); i++ {
		links[i] = strings.ToLower(links[i])
	}
	s.id++
	s.links[s.id] = links
	go func() {
		for key, value := range newLinks {
			s.cache.put(key, value)
		}
	}()
	return s.id, nil
}

func(s *Storage) Links(packageID int) (map[string]string, error) {
	links, ok := s.links[packageID]
	if !ok {
		return nil, errors.New("PackageNotFound")
	}
	res := make(map[string]string, len(links))
	for _, v := range links {
		res[v], _ = s.cache.get(v)
	}
	return res, nil
}

func(s *Storage) LinksStatus(links []string) map[string]string {
	res := make(map[string]string, len(links))
	for _, v := range links {
		status, ok := s.cache.get(strings.ToLower(v))
		if ok {
			res[v] = status
		}
	}

	return res
}

func(s *Storage) ValidateCache(newValues map[string]string) {
	for key, value := range newValues {
		s.cache.put(key, value)
	}
}

func(s *Storage) AllLinks() map[string]string {
	return s.cache.cache
}

func newCache(capacity int) *cache {
	if capacity <= 0 {
		panic("Cache capacity must be positive")
	}
	return &cache{
		cache:     make(map[string]string, capacity),
	}
}

func (c *cache) get(key string) (string, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	res, ok := c.cache[key]
	return res, ok
}

func (c *cache) put(key, value string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.cache[key] = value
}

type lruCache struct {
	capacity  int
	cache     map[string]*list.Element
	evictList *list.List
	mutex     sync.RWMutex
}

type entry struct {
	key   string
	value string
}

func NewLRUCache(capacity int) *lruCache {
	if capacity <= 0 {
		panic("LRU cache capacity must be positive")
	}
	return &lruCache{
		capacity:  capacity,
		cache:     make(map[string]*list.Element),
		evictList: list.New(),
	}
}

func (lru *lruCache) get(key string) (string, bool) {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	if elem, exists := lru.cache[key]; exists {
		lru.evictList.MoveToFront(elem)
		return elem.Value.(*entry).value, true
	}
	return "", false
}

func (lru *lruCache) put(key, value string) {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	if elem, exists := lru.cache[key]; exists {
		lru.evictList.MoveToFront(elem)
		elem.Value.(*entry).value = value
		return
	}

	if lru.evictList.Len() >= lru.capacity {
		lru.evict()
	}

	elem := lru.evictList.PushFront(&entry{key, value})
	lru.cache[key] = elem
}

func (lru *lruCache) evict() {
	elem := lru.evictList.Back()
	if elem != nil {
		lru.removeElement(elem)
	}
}

func (lru *lruCache) removeElement(elem *list.Element) {
	lru.evictList.Remove(elem)
	kv := elem.Value.(*entry)
	delete(lru.cache, kv.key)
}

func (lru *lruCache) remove(key string) bool {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	if elem, exists := lru.cache[key]; exists {
		lru.removeElement(elem)
		return true
	}
	return false
}

func (lru *lruCache) contains(key string) bool {
	lru.mutex.RLock()
	defer lru.mutex.RUnlock()
	
	_, exists := lru.cache[key]
	return exists
}

func (lru *lruCache) len() int {
	lru.mutex.RLock()
	defer lru.mutex.RUnlock()
	
	return lru.evictList.Len()
}

func (lru *lruCache) clear() {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()
	
	lru.cache = make(map[string]*list.Element)
	lru.evictList.Init()
}

func (lru *lruCache) keys() []string {
	lru.mutex.RLock()
	defer lru.mutex.RUnlock()
	
	keys := make([]string, 0, lru.evictList.Len())
	for elem := lru.evictList.Front(); elem != nil; elem = elem.Next() {
		keys = append(keys, elem.Value.(*entry).key)
	}
	return keys
}
