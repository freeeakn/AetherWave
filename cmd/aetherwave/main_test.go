package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/freeeakn/AetherWave/pkg/blockchain"
	"github.com/freeeakn/AetherWave/pkg/network"
)

func TestWithCORS(t *testing.T) {
	handler := withCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Проверяем CORS заголовки
	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected Access-Control-Allow-Origin: *")
	}

	if !strings.Contains(rr.Header().Get("Access-Control-Allow-Methods"), "GET") {
		t.Error("expected GET in Access-Control-Allow-Methods")
	}

	if !strings.Contains(rr.Header().Get("Access-Control-Allow-Headers"), "Authorization") {
		t.Error("expected Authorization in Access-Control-Allow-Headers")
	}
}

func TestWithCORSOptions(t *testing.T) {
	handler := withCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 for OPTIONS, got %d", rr.Code)
	}
}

func TestWithAuth(t *testing.T) {
	testToken := "test-token-123"
	handler := withAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true}`))
	}), testToken)

	tests := []struct {
		name       string
		path       string
		authHeader string
		wantStatus int
	}{
		{
			name:       "health endpoint without auth",
			path:       "/health",
			authHeader: "",
			wantStatus: http.StatusOK,
		},
		{
			name:       "ready endpoint without auth",
			path:       "/ready",
			authHeader: "",
			wantStatus: http.StatusOK,
		},
		{
			name:       "api without auth header",
			path:       "/api/test",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "api with invalid format",
			path:       "/api/test",
			authHeader: "Basic dGVzdA==",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "api with wrong token",
			path:       "/api/test",
			authHeader: "Bearer wrong-token",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "api with correct token",
			path:       "/api/test",
			authHeader: "Bearer " + testToken,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rr.Code)
			}
		})
	}
}

func TestWithRateLimit(t *testing.T) {
	handler := withRateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Первые запросы должны проходить
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("request %d: expected status 200, got %d", i, rr.Code)
		}
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b, want int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{1, 1, 1},
		{-1, 1, -1},
		{0, 0, 0},
	}

	for _, tt := range tests {
		got := min(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestStartAPIServer(t *testing.T) {
	bc := blockchain.NewBlockchain()
	node := network.NewNode(":3001", bc, nil)
	key := make([]byte, 32)
	name := "test"

	server := startAPIServer(":18080", bc, node, key, &name, "", "", "")

	if server == nil {
		t.Fatal("startAPIServer returned nil")
	}

	// Даем серверу время запуститься
	time.Sleep(100 * time.Millisecond)

	// Тестируем health endpoint
	resp, err := http.Get("http://localhost:18080/health")
	if err != nil {
		t.Fatalf("failed to get health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Тестируем ready endpoint
	resp2, err := http.Get("http://localhost:18080/ready")
	if err != nil {
		t.Fatalf("failed to get ready: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp2.StatusCode)
	}

	// Тестируем metrics endpoint
	resp3, err := http.Get("http://localhost:18080/metrics")
	if err != nil {
		t.Fatalf("failed to get metrics: %v", err)
	}
	defer resp3.Body.Close()

	if resp3.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp3.StatusCode)
	}

	// Тестируем blockchain endpoint
	resp4, err := http.Get("http://localhost:18080/api/blockchain")
	if err != nil {
		t.Fatalf("failed to get blockchain: %v", err)
	}
	defer resp4.Body.Close()

	if resp4.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp4.StatusCode)
	}

	// Тестируем deprecated GET /api/messages
	resp5, err := http.Get("http://localhost:18080/api/messages?username=test&key=1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	if err != nil {
		t.Fatalf("failed to get messages: %v", err)
	}
	defer resp5.Body.Close()

	if resp5.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for deprecated GET, got %d", resp5.StatusCode)
	}

	// Тестируем POST /api/messages
	body, _ := json.Marshal(map[string]string{
		"username": "test",
		"key":      "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
	})
	resp6, err := http.Post("http://localhost:18080/api/messages", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to post messages: %v", err)
	}
	defer resp6.Body.Close()

	if resp6.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 for POST, got %d", resp6.StatusCode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}

func TestStartAPIServerWithAuth(t *testing.T) {
	bc := blockchain.NewBlockchain()
	node := network.NewNode(":3002", bc, nil)
	key := make([]byte, 32)
	name := "test"
	token := "secret-token"

	server := startAPIServer(":18081", bc, node, key, &name, "", "", token)

	if server == nil {
		t.Fatal("startAPIServer returned nil")
	}

	// Даем серверу время запуститься
	time.Sleep(100 * time.Millisecond)

	// Запрос без токена должен быть отклонен
	resp, err := http.Get("http://localhost:18081/api/blockchain")
	if err != nil {
		t.Fatalf("failed to get blockchain: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401 without token, got %d", resp.StatusCode)
	}

	// Запрос с токеном должен проходить
	req, _ := http.NewRequest("GET", "http://localhost:18081/api/blockchain", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to get blockchain with token: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 with token, got %d", resp2.StatusCode)
	}

	// Health endpoint должен работать без токена
	resp3, err := http.Get("http://localhost:18081/health")
	if err != nil {
		t.Fatalf("failed to get health: %v", err)
	}
	defer resp3.Body.Close()

	if resp3.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 for health, got %d", resp3.StatusCode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}
