// Package sdk предоставляет SDK для интеграции с блокчейн-сетью AetherWave
package sdk

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ClientOptions содержит настройки для клиента AetherWave
type ClientOptions struct {
	NodeURL       string
	Timeout       time.Duration
	MaxRetries    int
	RetryDelay    time.Duration
	Username      string
	EncryptionKey string
}

// DefaultClientOptions возвращает настройки клиента по умолчанию
func DefaultClientOptions() ClientOptions {
	return ClientOptions{
		NodeURL:    "http://localhost:3000",
		Timeout:    10 * time.Second,
		MaxRetries: 3,
		RetryDelay: 1 * time.Second,
	}
}

// Client представляет клиент SDK для взаимодействия с AetherWave
type Client struct {
	options    ClientOptions
	httpClient *http.Client
}

// Message представляет сообщение в блокчейне
type Message struct {
	Sender     string `json:"sender"`
	Recipient  string `json:"recipient"`
	Content    string `json:"content"`
	Timestamp  int64  `json:"timestamp"`
	TimeString string `json:"timeString,omitempty"`
}

// Block представляет блок в блокчейне
type Block struct {
	Index     int       `json:"index"`
	Timestamp int64     `json:"timestamp"`
	Messages  []Message `json:"messages"`
	PrevHash  string    `json:"prevHash"`
	Hash      string    `json:"hash"`
	Nonce     int       `json:"nonce"`
}

// BlockchainInfo представляет информацию о блокчейне
type BlockchainInfo struct {
	BlockCount   int   `json:"blockCount"`
	MessageCount int   `json:"messageCount"`
	Difficulty   int   `json:"difficulty"`
	LastBlock    Block `json:"lastBlock"`
}

// PeerInfo представляет информацию о пире в сети
type PeerInfo struct {
	Address  string    `json:"address"`
	LastSeen time.Time `json:"lastSeen"`
	Active   bool      `json:"active"`
}

// ErrorResponse представляет ответ с ошибкой от API
type ErrorResponse struct {
	Error string `json:"error"`
}

// NewClient создает новый экземпляр клиента SDK
func NewClient(options ClientOptions) *Client {
	// Если опции не указаны, используем значения по умолчанию
	if options.NodeURL == "" {
		options.NodeURL = "http://localhost:3000"
	}
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
		options: options,
		httpClient: &http.Client{
			Timeout: options.Timeout,
		},
	}
}

// SetEncryptionKey устанавливает ключ шифрования для клиента
func (c *Client) SetEncryptionKey(key string) error {
	keyBytes, err := hex.DecodeString(key)
	if err != nil || len(keyBytes) != 32 {
		return fmt.Errorf("invalid hex-encoded 32-byte key: %v", err)
	}

	c.options.EncryptionKey = key
	return nil
}

// SetUsername устанавливает имя пользователя для клиента
func (c *Client) SetUsername(username string) {
	c.options.Username = username
}

// SendMessage отправляет сообщение получателю через блокчейн
func (c *Client) SendMessage(recipient, content string) error {
	if c.options.EncryptionKey == "" {
		return fmt.Errorf("encryption key is not set")
	}
	if c.options.Username == "" {
		return fmt.Errorf("username is not set")
	}

	// Создаем запрос
	data := map[string]string{
		"sender":    c.options.Username,
		"recipient": recipient,
		"content":   content,
		"key":       c.options.EncryptionKey,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to encode request: %v", err)
	}

	// Отправляем запрос
	resp, err := c.doRequest("POST", "/api/message", jsonData)
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		errorResp := parseErrorResponse(resp.Body)
		return fmt.Errorf("server returned error: %s (status: %d)", errorResp.Error, resp.StatusCode)
	}

	return nil
}

// GetMessages получает сообщения для текущего пользователя
func (c *Client) GetMessages() ([]Message, error) {
	if c.options.EncryptionKey == "" {
		return nil, fmt.Errorf("encryption key is not set")
	}
	if c.options.Username == "" {
		return nil, fmt.Errorf("username is not set")
	}

	// Создаем URL с параметрами
	endpoint := fmt.Sprintf("/api/messages?username=%s&key=%s",
		url.QueryEscape(c.options.Username),
		url.QueryEscape(c.options.EncryptionKey))

	// Отправляем запрос
	resp, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %v", err)
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		errorResp := parseErrorResponse(resp.Body)
		return nil, fmt.Errorf("server returned error: %s (status: %d)", errorResp.Error, resp.StatusCode)
	}

	// Разбираем ответ
	var messages []Message
	if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// Добавляем строковое представление времени
	for i := range messages {
		t := time.Unix(messages[i].Timestamp, 0)
		messages[i].TimeString = t.Format(time.RFC3339)
	}

	return messages, nil
}

// GetBlockchainInfo получает информацию о блокчейне
func (c *Client) GetBlockchainInfo() (*BlockchainInfo, error) {
	// Отправляем запрос
	resp, err := c.doRequest("GET", "/api/blockchain", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get blockchain info: %v", err)
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		errorResp := parseErrorResponse(resp.Body)
		return nil, fmt.Errorf("server returned error: %s (status: %d)", errorResp.Error, resp.StatusCode)
	}

	// Разбираем ответ
	var info BlockchainInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &info, nil
}

// GetPeers получает список пиров в сети
func (c *Client) GetPeers() ([]PeerInfo, error) {
	// Отправляем запрос
	resp, err := c.doRequest("GET", "/api/peers", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get peers: %v", err)
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		errorResp := parseErrorResponse(resp.Body)
		return nil, fmt.Errorf("server returned error: %s (status: %d)", errorResp.Error, resp.StatusCode)
	}

	// Разбираем ответ
	var peers []PeerInfo
	if err := json.NewDecoder(resp.Body).Decode(&peers); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return peers, nil
}

// GenerateEncryptionKey генерирует новый ключ шифрования
func (c *Client) GenerateEncryptionKey() (string, error) {
	// Отправляем запрос
	resp, err := c.doRequest("GET", "/api/generate-key", nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate key: %v", err)
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		errorResp := parseErrorResponse(resp.Body)
		return "", fmt.Errorf("server returned error: %s (status: %d)", errorResp.Error, resp.StatusCode)
	}

	// Разбираем ответ
	var result struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	// Сохраняем сгенерированный ключ
	c.options.EncryptionKey = result.Key

	return result.Key, nil
}

// doRequest выполняет HTTP запрос с повторными попытками
func (c *Client) doRequest(method, endpoint string, body []byte) (*http.Response, error) {
	url := c.options.NodeURL + endpoint

	var lastErr error
	for retry := 0; retry <= c.options.MaxRetries; retry++ {
		if retry > 0 {
			time.Sleep(c.options.RetryDelay)
		}

		var reqBody io.Reader
		if body != nil {
			reqBody = strings.NewReader(string(body))
		}

		req, err := http.NewRequest(method, url, reqBody)
		if err != nil {
			return nil, err
		}

		if method == "POST" {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.httpClient.Do(req)
		if err == nil {
			return resp, nil
		}

		lastErr = err
	}

	return nil, fmt.Errorf("request failed after %d retries: %v", c.options.MaxRetries, lastErr)
}

// parseErrorResponse разбирает ответ с ошибкой
func parseErrorResponse(body io.Reader) ErrorResponse {
	var errorResp ErrorResponse
	if err := json.NewDecoder(body).Decode(&errorResp); err != nil {
		errorResp.Error = "unknown error"
	}
	return errorResp
}
