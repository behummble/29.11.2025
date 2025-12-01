package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"
	"errors"

	"github.com/behummble/29-11-2025/internal/models"
)

type mockStorage struct {
	links    map[int][]string
	cache    map[string]string
	lastID   int
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		links: make(map[int][]string),
		cache:    make(map[string]string),
		lastID:   0,
	}
}

func (m *mockStorage) WriteLinksPackage(links []string) (int, error) {
	m.lastID++
	m.links[m.lastID] = links
	return m.lastID, nil
}

func (m *mockStorage) Links(packageID int) (map[string]string, []string, error) {
	links, exists := m.links[packageID]
	if !exists {
		return nil, nil, errors.New("PackageNotFound")
	}
	
	result := make(map[string]string)
	notCached := []string{}
	
	for _, link := range links {
		if status, exists := m.cache[link]; exists {
			result[link] = status
		} else {
			notCached = append(notCached, link)
		}
	}
	
	return result, notCached, nil
}

func (m *mockStorage) LinksStatus(links []string) map[string]string {
	result := make(map[string]string)
	for _, link := range links {
		if status, exists := m.cache[link]; exists {
			result[link] = status
		}
	}
	return result
}

func (m *mockStorage) ValidateCache(newValues map[string]string) {
	// Not needed for basic tests
}

func (m *mockStorage) AllLinks() map[string]string {
	result := make(map[string]string)
	for k, v := range m.cache {
		result[k] = v
	}
	return result
}

func (m *mockStorage) UpdateLinksInfo(links map[string]string) {
	for k, v := range links {
		m.cache[k] = v
	}
}

func TestLinkService_VerifyLinks(t *testing.T) {
	mockStorage := newMockStorage()
	service := NewService(mockStorage, slog.Default())
	
	links := []string{"example.com", "google.com"}
	request := models.VerifyLinksRequest{Links: links}
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	
	ctx := context.Background()
	response, err := service.VerifyLinks(ctx, data)
	if err != nil {
		t.Fatalf("VerifyLinks failed: %v", err)
	}
	
	if response.Links_num != 1 {
		t.Errorf("Expected Links_num 1, got %d", response.Links_num)
	}
	
	if len(response.Links) != 2 {
		t.Errorf("Expected 2 links in response, got %d", len(response.Links))
	}
}

func TestLinkService_VerifyLinks_EmptyBody(t *testing.T) {
	mockStorage := newMockStorage()
	service := NewService(mockStorage, slog.Default())
	
	ctx := context.Background()
	request := models.VerifyLinksRequest{}
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	_, err = service.VerifyLinks(ctx, data)
	
	if err == nil {
		t.Error("Expected error for empty body")
	}
	
	if err.Error() != "EmptyBody" {
		t.Errorf("Expected 'EmptyBody', got '%v'", err)
	}
}

func TestLinkService_VerifyLinks_InvalidJSON(t *testing.T) {
	mockStorage := newMockStorage()
	service := NewService(mockStorage, slog.Default())
	
	ctx := context.Background()
	_, err := service.VerifyLinks(ctx, []byte("{invalid json"))
	
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestLinkService_PackageLinks(t *testing.T) {
	mockStorage := newMockStorage()
	service := NewService(mockStorage, slog.Default())
	
	packageID, err := mockStorage.WriteLinksPackage([]string{"example.com"})
	if err != nil {
		t.Fatalf("WriteLinksPackage failed: %v", err)
	}
	
	request := models.LinksPackageRequest{Links_list: []int{packageID}}
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	
	ctx := context.Background()
	response, err := service.PackageLinks(ctx, data)
	if err != nil {
		t.Fatalf("PackageLinks failed: %v", err)
	}
	
	if len(response) == 0 {
		t.Error("Expected non-empty response")
	}
}

func TestLinkService_Shutdown(t *testing.T) {
	mockStorage := newMockStorage()
	service := NewService(mockStorage, slog.Default())
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go service.ValidateCache()
	time.Sleep(5 * time.Second)
	err := service.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}