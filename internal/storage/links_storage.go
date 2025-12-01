package storage

import (
	"container/list"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/behummble/29-11-2025/internal/config"
)

type Storage struct {
	links map[int][]string
	cache  *lruCache
	log *slog.Logger
	id int
}

func NewStorage(cfg config.StorageConfig, log *slog.Logger) *Storage {
	return &Storage{
		links: make(map[int][]string, cfg.LinksSize),
		cache: newLRUCache(cfg.CacheSize),
		log: log,
	}
}

func(s *Storage) WriteLinksPackage(links []string) (int, error) {
	for i := 0; i < len(links); i++ {
		links[i] = strings.ToLower(links[i])
	}
	s.id++
	s.links[s.id] = links
	return s.id, nil
}

func(s *Storage) Links(packageID int) (map[string]string, []string, error) {
	links, ok := s.links[packageID]
	if !ok {
		return nil, nil, fmt.Errorf("PackageNotFound: %d", packageID)
	}
	res := make(map[string]string, len(links))
	notInCache := make([]string, 0, len(links))
	for _, link := range links {
		value, ok := s.cache.get(link)
		if !ok {
			notInCache = append(notInCache, link)
		}
		res[link] = value
	}
	return res, notInCache, nil
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
	return s.cache.allKeys()
}

func(s *Storage) UpdateLinksInfo(links map[string]string) {
	for key, value := range links {
		s.cache.put(key, value)
	}
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

func newLRUCache(capacity int) *lruCache {
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

func (lru *lruCache) len() int {
	lru.mutex.RLock()
	defer lru.mutex.RUnlock()
	
	return lru.evictList.Len()
}

func (lru *lruCache) allKeys() map[string]string {
	res := make(map[string]string, lru.len())
	for key, elem := range lru.cache {
		res[key] = elem.Value.(*entry).value
	}
	return res
}