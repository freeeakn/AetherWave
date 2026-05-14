package sdk

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/freeeakn/AetherWave/pkg/crypto"
)

type ClientOptions struct {
	NodeURL    string
	Timeout    time.Duration
	MaxRetries int
	RetryDelay time.Duration
}

type Message struct {
	Sender     string `json:"sender"`
	Recipient  string `json:"recipient"`
	Content    string `json:"content"`
	TimeString string `json:"timeString,omitempty"`
	Timestamp  int64  `json:"timestamp,omitempty"`
	Encrypted  bool   `json:"encrypted,omitempty"`
}

type BlockInfo struct {
	Index     int       `json:"index"`
	Hash      string    `json:"hash"`
	PrevHash  string    `json:"prevHash"`
	Timestamp int64     `json:"timestamp"`
	Nonce     int       `json:"nonce"`
	Messages  []Message `json:"messages"`
}

type BlockchainInfo struct {
	BlockCount   int       `json:"blockCount"`
	MessageCount int       `json:"messageCount"`
	Difficulty   int       `json:"difficulty"`
	LastBlock    BlockInfo `json:"lastBlock"`
}

type PeerInfo struct {
	Address  string    `json:"address"`
	LastSeen time.Time `json:"lastSeen"`
	Active   bool      `json:"active"`
}

type Client struct {
	nodeURL       string
	httpClient    *http.Client
	encryptionKey []byte
	username      string
	maxRetries    int
	retryDelay    time.Duration
}

func NewClient(options ClientOptions) *Client {
	if options.Timeout == 0 {
		options.Timeout = 10 * time.Second
	}
	if options.MaxRetries == 0 {
		options.MaxRetries = 3
	}
	if options.RetryDelay == 0 {
		options.RetryDelay = 1 * time.Second
	}

	return &Client{
		nodeURL: options.NodeURL,
		httpClient: &http.Client{
			Timeout: options.Timeout,
		},
		maxRetries: options.MaxRetries,
		retryDelay: options.RetryDelay,
	}
}

func (c *Client) SetEncryptionKey(key string) error {
	keyBytes, err := hex.DecodeString(key)
	if err != nil || len(keyBytes) != 32 {
		return fmt.Errorf("invalid hex-encoded 32-byte key: %v", err)
	}
	c.encryptionKey = keyBytes
	return nil
}

func (c *Client) SetUsername(username string) {
	c.username = username
}

func (c *Client) GetUsername() string {
	return c.username
}

func (c *Client) GenerateEncryptionKey() (string, error) {
	key, err := crypto.GenerateKey()
	if err != nil {
		return "", fmt.Errorf("error generating key: %v", err)
	}
	c.encryptionKey = key
	return hex.EncodeToString(key), nil
}

func (c *Client) GetMessages() ([]Message, error) {
	if c.username == "" {
		return nil, errors.New("username not set")
	}

	endpoint := fmt.Sprintf("/api/messages?username=%s", c.username)
	var messages []Message

	err := c.makeRequest("GET", endpoint, nil, &messages)
	if err != nil {
		return nil, fmt.Errorf("error getting messages: %v", err)
	}

	return messages, nil
}

func (c *Client) SendMessage(recipient, content string) error {
	if c.username == "" {
		return errors.New("username not set")
	}

	if c.encryptionKey == nil {
		return errors.New("encryption key not set")
	}

	payload := struct {
		Sender    string `json:"sender"`
		Recipient string `json:"recipient"`
		Content   string `json:"content"`
		Key       string `json:"key"`
	}{
		Sender:    c.username,
		Recipient: recipient,
		Content:   content,
		Key:       hex.EncodeToString(c.encryptionKey),
	}

	err := c.makeRequest("POST", "/api/message", payload, nil)
	if err != nil {
		return fmt.Errorf("error sending message: %v", err)
	}

	return nil
}

func (c *Client) GetBlockchainInfo() (BlockchainInfo, error) {
	var info BlockchainInfo
	err := c.makeRequest("GET", "/api/blockchain", nil, &info)
	if err != nil {
		return BlockchainInfo{}, fmt.Errorf("error getting blockchain info: %v", err)
	}
	return info, nil
}

func (c *Client) GetPeers() ([]PeerInfo, error) {
	var peers []PeerInfo
	err := c.makeRequest("GET", "/api/peers", nil, &peers)
	if err != nil {
		return nil, fmt.Errorf("error getting peers: %v", err)
	}
	return peers, nil
}

func (c *Client) AddPeer(address string) error {
	peer := struct {
		Address string `json:"address"`
	}{
		Address: address,
	}

	err := c.makeRequest("POST", "/api/peers", peer, nil)
	if err != nil {
		return fmt.Errorf("error adding peer: %v", err)
	}
	return nil
}

func (c *Client) makeRequest(method, endpoint string, body interface{}, result interface{}) error {
	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("error marshaling JSON: %w", err)
		}
	}

	url := c.nodeURL + endpoint

	var lastErr error
	for retry := 0; retry <= c.maxRetries; retry++ {
		var req *http.Request
		var err error

		if reqBody != nil {
			req, err = http.NewRequest(method, url, bytes.NewReader(reqBody))
		} else {
			req, err = http.NewRequest(method, url, nil)
		}
		if err != nil {
			return fmt.Errorf("error creating request: %w", err)
		}

		if method == "POST" {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.httpClient.Do(req)
		if err == nil {
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("API error: %s (code %d)", string(bodyBytes), resp.StatusCode)
			}

			if result != nil {
				bodyBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					return fmt.Errorf("error reading response: %w", err)
				}
				if err := json.Unmarshal(bodyBytes, result); err != nil {
					return fmt.Errorf("error unmarshaling JSON: %w", err)
				}
			}
			return nil
		}
		lastErr = err
		if retry < c.maxRetries {
			time.Sleep(c.retryDelay)
		}
	}

	return fmt.Errorf("request failed after %d retries: %w", c.maxRetries, lastErr)
}
