package sdk

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient_DefaultOptions(t *testing.T) {
	c := NewClient(ClientOptions{})
	if c.options.NodeURL != "http://localhost:3000" {
		t.Fatalf("Expected default NodeURL 'http://localhost:3000', got '%s'", c.options.NodeURL)
	}
	if c.options.MaxRetries != 3 {
		t.Fatalf("Expected default MaxRetries 3, got %d", c.options.MaxRetries)
	}
}

func TestSetEncryptionKey_ValidHex(t *testing.T) {
	c := NewClient(ClientOptions{})
	err := c.SetEncryptionKey("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("Expected valid hex key, got error: %v", err)
	}
	if c.options.EncryptionKey != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Fatalf("Expected key stored as-is, got '%s'", c.options.EncryptionKey)
	}
}

func TestSetEncryptionKey_InvalidHex(t *testing.T) {
	c := NewClient(ClientOptions{})
	err := c.SetEncryptionKey("not-hex!")
	if err == nil {
		t.Fatal("Expected error for invalid hex key")
	}
}

func TestSetEncryptionKey_ShortHex(t *testing.T) {
	c := NewClient(ClientOptions{})
	err := c.SetEncryptionKey("aabb")
	if err == nil {
		t.Fatal("Expected error for short key (must be 32 bytes = 64 hex chars)")
	}
}

func TestSetUsername(t *testing.T) {
	c := NewClient(ClientOptions{})
	c.SetUsername("Alice")
	if c.options.Username != "Alice" {
		t.Fatalf("Expected username 'Alice', got '%s'", c.options.Username)
	}
}

func TestSendMessage_NoKey(t *testing.T) {
	c := NewClient(ClientOptions{Username: "Alice"})
	err := c.SendMessage("Bob", "Hello")
	if err == nil {
		t.Fatal("Expected error when key not set")
	}
}

func TestSendMessage_NoUsername(t *testing.T) {
	c := NewClient(ClientOptions{EncryptionKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"})
	err := c.SendMessage("Bob", "Hello")
	if err == nil {
		t.Fatal("Expected error when username not set")
	}
}

func TestGetMessages_NoKey(t *testing.T) {
	c := NewClient(ClientOptions{Username: "Alice"})
	_, err := c.GetMessages()
	if err == nil {
		t.Fatal("Expected error when key not set")
	}
}

func TestGetMessages_NoUsername(t *testing.T) {
	c := NewClient(ClientOptions{EncryptionKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"})
	_, err := c.GetMessages()
	if err == nil {
		t.Fatal("Expected error when username not set")
	}
}

func TestSendMessage_Integration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("Expected POST, got %s", r.Method)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("Failed to decode body: %v", err)
		}
		if body["sender"] != "Alice" {
			t.Fatalf("Expected sender 'Alice', got '%s'", body["sender"])
		}
		if body["key"] != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
			t.Fatalf("Expected hex key in body, got '%s'", body["key"])
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}))
	defer server.Close()

	c := NewClient(ClientOptions{
		NodeURL:       server.URL,
		Username:      "Alice",
		EncryptionKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	})
	err := c.SendMessage("Bob", "Hello Bob!")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func TestGetMessages_Integration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("Expected POST, got %s", r.Method)
		}
		var body struct {
			Username string `json:"username"`
			Key      string `json:"key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("Failed to decode body: %v", err)
		}
		if body.Username != "Alice" {
			t.Fatalf("Expected username=Alice, got '%s'", body.Username)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"sender": "Bob", "recipient": "Alice", "content": "Hello!", "timestamp": 1000000},
		})
	}))
	defer server.Close()

	c := NewClient(ClientOptions{
		NodeURL:       server.URL,
		Username:      "Alice",
		EncryptionKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	})
	msgs, err := c.GetMessages()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Content != "Hello!" {
		t.Fatalf("Expected content 'Hello!', got '%s'", msgs[0].Content)
	}
}

func TestGetBlockchainInfo_Integration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(BlockchainInfo{
			BlockCount: 3, MessageCount: 7, Difficulty: 4,
			LastBlock: Block{Index: 2, Hash: "0000abc", Nonce: 42},
		})
	}))
	defer server.Close()

	c := NewClient(ClientOptions{NodeURL: server.URL})
	info, err := c.GetBlockchainInfo()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if info.BlockCount != 3 {
		t.Fatalf("Expected BlockCount 3, got %d", info.BlockCount)
	}
	if info.LastBlock.Index != 2 {
		t.Fatalf("Expected LastBlock.Index 2, got %d", info.LastBlock.Index)
	}
}

func TestGetPeers_Integration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]PeerInfo{
			{Address: "10.0.0.1:8080", Active: true},
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
}

func TestGenerateEncryptionKey_Integration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"key": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		})
	}))
	defer server.Close()

	c := NewClient(ClientOptions{NodeURL: server.URL})
	key, err := c.GenerateEncryptionKey()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if key != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Fatalf("Expected generated key, got '%s'", key)
	}
}

func TestDefaultClientOptions(t *testing.T) {
	opts := DefaultClientOptions()
	if opts.NodeURL != "http://localhost:3000" {
		t.Fatalf("Expected NodeURL 'http://localhost:3000', got '%s'", opts.NodeURL)
	}
}
