package sdk

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetEncryptionKey_ValidHex(t *testing.T) {
	c := NewClient(ClientOptions{})
	err := c.SetEncryptionKey("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("Expected valid hex key, got error: %v", err)
	}
	if len(c.encryptionKey) != 32 {
		t.Fatalf("Expected key length 32, got %d", len(c.encryptionKey))
	}
}

func TestSetEncryptionKey_InvalidHex(t *testing.T) {
	c := NewClient(ClientOptions{})
	err := c.SetEncryptionKey("not-hex")
	if err == nil {
		t.Fatal("Expected error for invalid hex key")
	}
}

func TestSetEncryptionKey_ShortKey(t *testing.T) {
	c := NewClient(ClientOptions{})
	err := c.SetEncryptionKey("aabb")
	if err == nil {
		t.Fatal("Expected error for short key (must be 32 bytes)")
	}
}

func TestSetUsername(t *testing.T) {
	c := NewClient(ClientOptions{})
	c.SetUsername("Alice")
	if c.GetUsername() != "Alice" {
		t.Fatalf("Expected username 'Alice', got '%s'", c.GetUsername())
	}
}

func TestGenerateEncryptionKey(t *testing.T) {
	c := NewClient(ClientOptions{})
	keyStr, err := c.GenerateEncryptionKey()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(keyStr) != 64 {
		t.Fatalf("Expected hex key of length 64, got %d", len(keyStr))
	}
	if len(c.encryptionKey) != 32 {
		t.Fatalf("Expected internal key of length 32, got %d", len(c.encryptionKey))
	}
}

func TestSendMessage_NoUsername(t *testing.T) {
	c := NewClient(ClientOptions{})
	err := c.SendMessage("Bob", "Hello")
	if err == nil {
		t.Fatal("Expected error when username not set")
	}
}

func TestGetMessages_NoUsername(t *testing.T) {
	c := NewClient(ClientOptions{})
	_, err := c.GetMessages()
	if err == nil {
		t.Fatal("Expected error when username not set")
	}
}

func TestSendMessage_NoKey(t *testing.T) {
	c := NewClient(ClientOptions{})
	c.SetUsername("Alice")
	err := c.SendMessage("Bob", "Hello")
	if err == nil {
		t.Fatal("Expected error when key not set")
	}
}

func TestGetMessages_Integration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/messages" {
			t.Fatalf("Expected /api/messages, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("username") != "Alice" {
			t.Fatalf("Expected username=Alice, got %s", r.URL.Query().Get("username"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Message{
			{Sender: "Bob", Recipient: "Alice", Content: "Hello Alice!", Timestamp: 1000000},
		})
	}))
	defer server.Close()

	c := NewClient(ClientOptions{NodeURL: server.URL})
	c.SetUsername("Alice")
	err := c.SetEncryptionKey("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}

	msgs, err := c.GetMessages()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Content != "Hello Alice!" {
		t.Fatalf("Expected 'Hello Alice!', got '%s'", msgs[0].Content)
	}
}

func TestSendMessage_Integration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("Expected POST, got %s", r.Method)
		}
		var body struct {
			Sender    string `json:"sender"`
			Recipient string `json:"recipient"`
			Content   string `json:"content"`
			Key       string `json:"key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("Failed to decode body: %v", err)
		}
		if body.Sender != "Alice" {
			t.Fatalf("Expected sender Alice, got %s", body.Sender)
		}
		if body.Key != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
			t.Fatalf("Expected hex key in request, got %s", body.Key)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}))
	defer server.Close()

	c := NewClient(ClientOptions{NodeURL: server.URL})
	c.SetUsername("Alice")
	err := c.SetEncryptionKey("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}

	err = c.SendMessage("Bob", "Hello Bob!")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func TestGetBlockchainInfo_Integration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(BlockchainInfo{
			BlockCount: 5, MessageCount: 10, Difficulty: 4,
		})
	}))
	defer server.Close()

	c := NewClient(ClientOptions{NodeURL: server.URL})
	info, err := c.GetBlockchainInfo()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if info.BlockCount != 5 {
		t.Fatalf("Expected BlockCount 5, got %d", info.BlockCount)
	}
}

func TestGetPeers_Integration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]PeerInfo{
			{Address: "192.168.1.1:8080", Active: true},
		})
	}))
	defer server.Close()

	c := NewClient(ClientOptions{NodeURL: server.URL})
	peers, err := c.GetPeers()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(peers) != 1 {
		t.Fatalf("Expected 1 peer, got %d", len(peers))
	}
	if peers[0].Address != "192.168.1.1:8080" {
		t.Fatalf("Expected address '192.168.1.1:8080', got '%s'", peers[0].Address)
	}
}

func TestAddPeer_Integration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("Expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}))
	defer server.Close()

	c := NewClient(ClientOptions{NodeURL: server.URL})
	err := c.AddPeer("192.168.1.2:8080")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func TestMakeRequest_RetriesOnNetworkError(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Fatal("server doesn't support hijacking")
			}
			conn, _, _ := hj.Hijack()
			conn.Close()
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	c := NewClient(ClientOptions{NodeURL: server.URL, MaxRetries: 3, RetryDelay: 1})
	var result map[string]string
	err := c.makeRequest("GET", "/test", nil, &result)
	if err != nil {
		t.Fatalf("Expected no error after retries, got: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("Expected 3 attempts, got %d", attempts)
	}
}
