package storage

import (
	"log/slog"
	"testing"

	"github.com/behummble/29-11-2025/internal/config"
)

func TestStorage_WriteAndReadLinksPackage(t *testing.T) {
	cfg := config.StorageConfig{LinksSize: 100, CacheSize: 50}
	storage := NewStorage(cfg, slog.Default())
	
	links := []string{"example.com", "google.com"}
	
	id, err := storage.WriteLinksPackage(links)
	if err != nil {
		t.Fatalf("WriteLinksPackage failed: %v", err)
	}
	
	if id != 1 {
		t.Errorf("Expected ID 1, got %d", id)
	}
	
	cached, notCached, err := storage.Links(id)
	if err != nil {
		t.Fatalf("Links failed: %v", err)
	}
	
	if len(cached) != 2 {
		t.Errorf("Expected 2 cached links, got %d", len(cached))
	}
	
	if len(notCached) != 2 {
		t.Errorf("Expected 2 not cached links, got %d", len(notCached))
	}
	
	storage.UpdateLinksInfo(map[string]string{
		"example.com": "available",
		"google.com":  "available",
	})
	
	cached, notCached, err = storage.Links(id)
	if err != nil {
		t.Fatalf("Links failed: %v", err)
	}
	
	if len(cached) != 2 {
		t.Errorf("Expected 2 cached links after update, got %d", len(cached))
	}
	
	if len(notCached) != 0 {
		t.Errorf("Expected 0 not cached links after update, got %d", len(notCached))
	}
}

func TestStorage_LinksStatus(t *testing.T) {
	cfg := config.StorageConfig{LinksSize: 100, CacheSize: 50}
	storage := NewStorage(cfg, slog.Default())
	
	storage.UpdateLinksInfo(map[string]string{
		"cached.com": "available",
	})
	
	status := storage.LinksStatus([]string{"cached.com", "uncached.com"})
	
	if len(status) != 1 {
		t.Errorf("Expected 1 status, got %d", len(status))
	}
	
	if status["cached.com"] != "available" {
		t.Errorf("Expected 'available', got '%s'", status["cached.com"])
	}
}

func TestStorage_PackageNotFound(t *testing.T) {
	cfg := config.StorageConfig{LinksSize: 100, CacheSize: 50}
	storage := NewStorage(cfg, slog.Default())
	
	_, _, err := storage.Links(999)
	
	if err == nil {
		t.Error("Expected error for non-existent package")
	}
	
	if err.Error() != "PackageNotFound: 999" {
		t.Errorf("Expected 'PackageNotFound: 999', got '%v'", err)
	}
}

func TestLRUCache_Eviction(t *testing.T) {
	cache := newLRUCache(2)
	
	cache.put("key1", "value1")
	cache.put("key2", "value2")
	cache.put("key3", "value3")
	
	_, ok := cache.get("key1")
	if ok {
		t.Error("Expected key1 to be evicted")
	}
	
	val, ok := cache.get("key2")
	if !ok {
		t.Error("Expected key2 to be present")
	}
	if val != "value2" {
		t.Errorf("Expected 'value2', got '%s'", val)
	}
	
	val, ok = cache.get("key3")
	if !ok {
		t.Error("Expected key3 to be present")
	}
	if val != "value3" {
		t.Errorf("Expected 'value3', got '%s'", val)
	}
}