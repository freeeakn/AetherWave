// Package mobile содержит примеры использования SDK AetherWave в мобильных приложениях
package mobile

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/freeeakn/AetherWave/sdk"
)

// AetherWaveMobile представляет обертку SDK для использования в мобильных приложениях
type AetherWaveMobile struct {
	client     *sdk.Client
	isLoggedIn bool
	username   string
}

// MessageData представляет сообщение в формате для мобильного приложения
type MessageData struct {
	ID         string `json:"id"`
	Sender     string `json:"sender"`
	Recipient  string `json:"recipient"`
	Content    string `json:"content"`
	Timestamp  string `json:"timestamp"`
	IsOutgoing bool   `json:"isOutgoing"`
}

// BlockchainInfoData представляет информацию о блокчейне в формате для мобильного приложения
type BlockchainInfoData struct {
	BlockCount    int    `json:"blockCount"`
	MessageCount  int    `json:"messageCount"`
	Difficulty    int    `json:"difficulty"`
	LastBlockID   int    `json:"lastBlockId"`
	LastBlockHash string `json:"lastBlockHash"`
}

// NewAetherWaveMobile создает новый экземпляр обертки SDK для мобильных приложений
func NewAetherWaveMobile(nodeURL string) *AetherWaveMobile {
	options := sdk.ClientOptions{
		NodeURL:    nodeURL,
		Timeout:    15 * time.Second,
		MaxRetries: 3,
		RetryDelay: 1 * time.Second,
	}

	return &AetherWaveMobile{
		client:     sdk.NewClient(options),
		isLoggedIn: false,
	}
}

// Login выполняет вход в систему
func (m *AetherWaveMobile) Login(username, encryptionKey string) string {
	m.client.SetUsername(username)

	// После обновления SDK метод возвращает ошибку
	err := m.client.SetEncryptionKey(encryptionKey)
	if err != nil {
		return formatErrorResponse(err)
	}

	m.isLoggedIn = true
	m.username = username
	return `{"success": true, "message": "Вход успешно выполнен"}`
}

// GenerateKey генерирует новый ключ шифрования
func (m *AetherWaveMobile) GenerateKey() string {
	key, err := m.client.GenerateEncryptionKey()
	if err != nil {
		return formatErrorResponse(err)
	}

	result := map[string]interface{}{
		"success": true,
		"key":     key,
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return formatErrorResponse(err)
	}

	return string(jsonData)
}

// GetMessages получает сообщения для текущего пользователя
func (m *AetherWaveMobile) GetMessages() string {
	if !m.isLoggedIn {
		return `{"success": false, "error": "User not logged in"}`
	}

	messages, err := m.client.GetMessages()
	if err != nil {
		return formatErrorResponse(err)
	}

	// Преобразуем сообщения в формат для мобильного приложения
	username := m.username
	messageDataList := make([]MessageData, len(messages))

	for i, msg := range messages {
		messageDataList[i] = MessageData{
			ID:         fmt.Sprintf("msg_%d", i),
			Sender:     msg.Sender,
			Recipient:  msg.Recipient,
			Content:    msg.Content,
			Timestamp:  msg.TimeString,
			IsOutgoing: msg.Sender == username,
		}
	}

	result := map[string]interface{}{
		"success":  true,
		"messages": messageDataList,
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return formatErrorResponse(err)
	}

	return string(jsonData)
}

// SendMessage отправляет сообщение
func (m *AetherWaveMobile) SendMessage(recipient, content string) string {
	if !m.isLoggedIn {
		return `{"success": false, "error": "User not logged in"}`
	}

	err := m.client.SendMessage(recipient, content)
	if err != nil {
		return formatErrorResponse(err)
	}

	return `{"success": true, "message": "Message sent successfully"}`
}

// GetBlockchainInfo получает информацию о блокчейне
func (m *AetherWaveMobile) GetBlockchainInfo() string {
	infoPtr, err := m.client.GetBlockchainInfo()
	if err != nil {
		return formatErrorResponse(err)
	}

	// Работаем с указателем, возвращаемым из SDK
	info := *infoPtr

	// Преобразуем информацию в формат для мобильного приложения
	infoData := BlockchainInfoData{
		BlockCount:    info.BlockCount,
		MessageCount:  info.MessageCount,
		Difficulty:    info.Difficulty,
		LastBlockID:   info.LastBlock.Index,
		LastBlockHash: info.LastBlock.Hash,
	}

	result := map[string]interface{}{
		"success":        true,
		"blockchainInfo": infoData,
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return formatErrorResponse(err)
	}

	return string(jsonData)
}

// GetPeers получает список пиров
func (m *AetherWaveMobile) GetPeers() string {
	peers, err := m.client.GetPeers()
	if err != nil {
		return formatErrorResponse(err)
	}

	// Преобразуем пиры в формат для мобильного приложения
	peersList := make([]map[string]interface{}, len(peers))

	for i, peer := range peers {
		peersList[i] = map[string]interface{}{
			"address":  peer.Address,
			"lastSeen": peer.LastSeen.Format(time.RFC3339),
			"active":   peer.Active,
		}
	}

	result := map[string]interface{}{
		"success": true,
		"peers":   peersList,
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return formatErrorResponse(err)
	}

	return string(jsonData)
}

// Helpers

// formatErrorResponse форматирует ответ с ошибкой
func formatErrorResponse(err error) string {
	result := map[string]interface{}{
		"success": false,
		"error":   err.Error(),
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return `{"success": false, "error": "Failed to format error response"}`
	}

	return string(jsonData)
}
