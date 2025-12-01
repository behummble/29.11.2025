package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/behummble/29-11-2025/internal/config"
	"github.com/behummble/29-11-2025/internal/models"
)

type mockService struct {
	verifyLinksResponse models.VerifyLinksResponse
	verifyLinksError    error
	packageLinksResponse []byte
	packageLinksError    error
}

func (m *mockService) VerifyLinks(ctx context.Context, data []byte) (models.VerifyLinksResponse, error) {
	return m.verifyLinksResponse, m.verifyLinksError
}

func (m *mockService) PackageLinks(ctx context.Context, data []byte) ([]byte, error) {
	return m.packageLinksResponse, m.packageLinksError
}

func TestServer_VerifyLinks_Success(t *testing.T) {
	mockService := &mockService{
		verifyLinksResponse: models.VerifyLinksResponse{
			Links: map[string]string{"example.com": "available"},
			Links_num: 1,
		},
	}
	
	server := NewServer(slog.Default(), config.ServerConfig{
		Host: "localhost",
		Port: 8080,
	}, mockService)
	
	requestBody := models.VerifyLinksRequest{
		Links: []string{"example.com"},
	}
	data, _ := json.Marshal(requestBody)
	
	req := httptest.NewRequest("POST", "/links", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	
	handler := server.GetHandler()
	handler.ServeHTTP(rr, req)
	
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
	
	var response models.VerifyLinksResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	
	if response.Links_num != 1 {
		t.Errorf("Expected Links_num 1, got %d", response.Links_num)
	}
	
	if response.Links["example.com"] != "available" {
		t.Errorf("Expected 'available', got '%s'", response.Links["example.com"])
	}
}

func TestServer_VerifyLinks_EmptyBody(t *testing.T) {
	mockService := &mockService{}
	server := NewServer(slog.Default(), config.ServerConfig{
		Host: "localhost",
		Port: 8080,
	}, mockService)
	
	req := httptest.NewRequest("POST", "/links", bytes.NewReader([]byte{}))
	rr := httptest.NewRecorder()
	
	handler := server.GetHandler()
	handler.ServeHTTP(rr, req)
	
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestServer_VerifyLinks_ServiceError(t *testing.T) {
	mockService := &mockService{
		verifyLinksError: errors.New("service error"),
	}
	server := NewServer(slog.Default(), config.ServerConfig{
		Host: "localhost",
		Port: 8080,
	}, mockService)
	
	requestBody := models.VerifyLinksRequest{
		Links: []string{"example.com"},
	}
	data, _ := json.Marshal(requestBody)
	
	req := httptest.NewRequest("POST", "/links", bytes.NewReader(data))
	rr := httptest.NewRecorder()
	
	handler := server.GetHandler()
	handler.ServeHTTP(rr, req)
	
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestServer_LinksReport(t *testing.T) {
	responseData, _ := json.Marshal(map[string]string{"example.com": "available"})
	mockService := &mockService{
		packageLinksResponse: responseData,
	}
	server := NewServer(slog.Default(), config.ServerConfig{
		Host: "localhost",
		Port: 8080,
	}, mockService)
	
	requestBody := models.LinksPackageRequest{
		Links_list: []int{1},
	}
	data, _ := json.Marshal(requestBody)
	
	req := httptest.NewRequest("POST", "/links/list", bytes.NewReader(data))
	rr := httptest.NewRecorder()
	
	handler := server.GetHandler()
	handler.ServeHTTP(rr, req)
	
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
	
	if rr.Header().Get("Content-type") != "application/pdf" {
		t.Errorf("Expected Content-type 'application/pdf', got '%s'", rr.Header().Get("Content-type"))
	}
}
