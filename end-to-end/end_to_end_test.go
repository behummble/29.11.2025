package endtoend

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/behummble/29-11-2025/internal/app"
	"github.com/behummble/29-11-2025/internal/config"
	"github.com/behummble/29-11-2025/internal/models"
)

func TestEndToEnd_LinkVerification(t *testing.T) {
	cfg := config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8081,
		},
		Storage: config.StorageConfig{
			LinksSize: 100,
			CacheSize: 50,
		},
		Log: config.LogConfig{
			Level: -4,
		},
	}
	
	testApp := app.NewApp(cfg, slog.Default())
	
	go testApp.Run()
	
	time.Sleep(500 * time.Millisecond)
	
	t.Run("VerifyLinks", func(t *testing.T) {
		request := models.VerifyLinksRequest{
			Links: []string{"httpbin.org/status/200", "httpbin.org/status/404"},
		}
		
		data, err := json.Marshal(request)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}
		
		resp, err := http.Post("http://localhost:8081/links", "application/json", bytes.NewReader(data))
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status %d, got %d", http.StatusCreated, resp.StatusCode)
		}
		
		var response models.VerifyLinksResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		
		if len(response.Links) != 2 {
			t.Errorf("Expected 2 links in response, got %d", len(response.Links))
		}
		
		if response.Links_num <= 0 {
			t.Errorf("Expected positive Links_num, got %d", response.Links_num)
		}
		
		if _, exists := response.Links["httpbin.org/status/200"]; !exists {
			t.Error("Missing status for httpbin.org/status/200")
		}
		if _, exists := response.Links["httpbin.org/status/404"]; !exists {
			t.Error("Missing status for httpbin.org/status/404")
		}
	})
	
	// Test 2: Get package links (using the ID from previous test)
	t.Run("PackageLinks", func(t *testing.T) {
		request := models.LinksPackageRequest{
			Links_list: []int{1}, // Use ID from first test
		}
		
		data, err := json.Marshal(request)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}
		
		resp, err := http.Post("http://localhost:8081/links/list", "application/json", bytes.NewReader(data))
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}
		
		if resp.Header.Get("Content-type") != "application/pdf" {
			t.Errorf("Expected Content-type 'application/pdf', got '%s'", resp.Header.Get("Content-type"))
		}
	})
	
	// Test 3: Invalid request
	t.Run("InvalidRequest", func(t *testing.T) {
		resp, err := http.Post("http://localhost:8081/links", "application/json", bytes.NewReader([]byte("invalid json")))
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, resp.StatusCode)
		}
	})
	
	//shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	//defer cancel()
	//testApp.Shutdown(shutdownCtx)
}

// Simple local server for testing without external dependencies
func TestEndToEnd_WithLocalServer(t *testing.T) {
	localServer := http.Server{
		Addr: ":8090",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/success":
				w.WriteHeader(http.StatusOK)
			case "/error":
				w.WriteHeader(http.StatusInternalServerError)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}),
	}
	
	go func() {
		localServer.ListenAndServe()
	}()
	defer localServer.Shutdown(context.Background())
	
	time.Sleep(100 * time.Millisecond)
	
	cfg := config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8082,
		},
		Storage: config.StorageConfig{
			LinksSize: 100,
			CacheSize: 50,
		},
		Log: config.LogConfig{
			Level: -4,
		},
	}
	
	testApp := app.NewApp(cfg, slog.Default())
	go testApp.Run()
	defer testApp.Shutdown(context.Background())
	
	time.Sleep(500 * time.Millisecond)
	
	request := models.VerifyLinksRequest{
		Links: []string{"localhost:8090/success", "localhost:8090/error"},
	}
	
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}
	
	resp, err := http.Post("http://localhost:8082/links", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}
	
	var response models.VerifyLinksResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if len(response.Links) != 2 {
		t.Errorf("Expected 2 links in response, got %d", len(response.Links))
	}

	for key, value := range response.Links {
		if key == "localhost:8090/success" && value != "avaliable" {
			t.Errorf("Expected that link %s will be avaliable, got %s", key, value)
		} else if key == "localhost:8090/error" && value == "avaliable" {
			t.Errorf("Expected that link %s will be not avaliable, got %s", key, value)
		}
	}
}