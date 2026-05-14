package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/freeeakn/AetherWave/pkg/blockchain"
)

// mockConn реализует интерфейс net.Conn для тестирования
type mockConn struct {
	readBuf  *bytes.Buffer
	writeBuf *bytes.Buffer
	closed   bool
	mu       sync.Mutex
}

func newMockConn() *mockConn {
	return &mockConn{
		readBuf:  new(bytes.Buffer),
		writeBuf: new(bytes.Buffer),
	}
}

func (c *mockConn) Read(b []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.readBuf.Read(b)
}

func (c *mockConn) Write(b []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.writeBuf.Write(b)
}

func (c *mockConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

func (c *mockConn) LocalAddr() net.Addr                { return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8080} }
func (c *mockConn) RemoteAddr() net.Addr               { return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8081} }
func (c *mockConn) SetDeadline(t time.Time) error      { return nil }
func (c *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *mockConn) SetWriteDeadline(t time.Time) error { return nil }

// mockListener реализует интерфейс net.Listener для тестирования
type mockListener struct {
	connCh chan net.Conn
	closed bool
	mu     sync.Mutex
}

func newMockListener() *mockListener {
	return &mockListener{
		connCh: make(chan net.Conn, 10),
	}
}

func (l *mockListener) Accept() (net.Conn, error) {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil, net.ErrClosed
	}
	l.mu.Unlock()
	conn, ok := <-l.connCh
	if !ok {
		return nil, net.ErrClosed
	}
	return conn, nil
}

func (l *mockListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.closed {
		l.closed = true
		close(l.connCh)
	}
	return nil
}

func (l *mockListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8080}
}

func (l *mockListener) AddConnection(conn net.Conn) {
	l.connCh <- conn
}

func TestNewNode(t *testing.T) {
	bc := blockchain.NewBlockchain()
	bootstrapPeers := []string{"192.168.1.1:8081"}
	node := NewNode("127.0.0.1:8080", bc, bootstrapPeers)

	if node == nil {
		t.Fatal("NewNode() returned nil")
	}

	if node.Address != "127.0.0.1:8080" {
		t.Errorf("Expected node address to be 127.0.0.1:8080, got %s", node.Address)
	}

	if node.Blockchain != bc {
		t.Errorf("Expected node blockchain to be the same as provided")
	}

	if len(node.Peers) != 0 {
		t.Errorf("Expected node to have 0 peers initially, got %d", len(node.Peers))
	}

	if len(node.bootstrapPeers) != 1 || node.bootstrapPeers[0] != "192.168.1.1:8081" {
		t.Errorf("Expected bootstrap peers to be [192.168.1.1:8081], got %v", node.bootstrapPeers)
	}
}

func TestConnectToPeer(t *testing.T) {
	bc := blockchain.NewBlockchain()
	node := NewNode("127.0.0.1:8080", bc, nil)

	// Создаем мок-соединение
	conn := newMockConn()

	// Заменяем функцию DialPeer для тестирования
	originalDialPeer := DialPeer
	defer func() { DialPeer = originalDialPeer }()

	dialCount := 0
	DialPeer = func(address string) (net.Conn, error) {
		dialCount++
		if address == "192.168.1.1:8081" {
			return conn, nil
		}
		return nil, fmt.Errorf("mock dial error")
	}

	// Подключаемся к пиру
	err := node.ConnectToPeer("192.168.1.1:8081")
	if err != nil {
		t.Fatalf("Failed to connect to peer: %v", err)
	}

	// Проверяем, что функция DialPeer была вызвана
	if dialCount != 1 {
		t.Errorf("Expected DialPeer to be called 1 time, got %d", dialCount)
	}

	// Проверяем, что пир был добавлен
	node.mutex.RLock()
	if len(node.Peers) != 1 {
		t.Errorf("Expected node to have 1 peer after connecting, got %d", len(node.Peers))
	}
	node.mutex.RUnlock()

	// Проверяем, что пир был добавлен в список известных пиров
	node.mutex.RLock()
	if len(node.KnownPeers) != 1 {
		t.Errorf("Expected node to have 1 known peer after connecting, got %d", len(node.KnownPeers))
	}

	peerInfo, exists := node.KnownPeers["192.168.1.1:8081"]
	node.mutex.RUnlock()

	if !exists {
		t.Errorf("Expected peer 192.168.1.1:8081 to be in known peers")
	}

	if !peerInfo.Active {
		t.Errorf("Expected peer 192.168.1.1:8081 to be active")
	}
}

func TestDisconnectPeer(t *testing.T) {
	bc := blockchain.NewBlockchain()
	node := NewNode("127.0.0.1:8080", bc, nil)

	// Создаем мок-соединение
	conn := newMockConn()

	// Добавляем пир вручную
	node.mutex.Lock()
	node.Peers["192.168.1.1:8081"] = conn
	node.KnownPeers["192.168.1.1:8081"] = PeerInfo{
		Address:  "192.168.1.1:8081",
		LastSeen: time.Now(),
		Active:   true,
	}
	node.mutex.Unlock()

	// Отключаем пир
	node.disconnectPeer("192.168.1.1:8081")

	// Проверяем, что пир был удален из активных соединений
	node.mutex.RLock()
	if len(node.Peers) != 0 {
		t.Errorf("Expected node to have 0 peers after disconnecting, got %d", len(node.Peers))
	}

	// Проверяем, что соединение было закрыто
	if !conn.closed {
		t.Errorf("Expected connection to be closed")
	}
}

func TestHandleConnection(t *testing.T) {
	bc := blockchain.NewBlockchain()
	node := NewNode("127.0.0.1:8080", bc, nil)

	// Создаем мок-соединение
	conn := newMockConn()

	// Создаем сообщение для отправки
	pingMsg := NetworkMessage{
		Type: "ping",
	}

	// Сериализуем сообщение и записываем в буфер чтения соединения
	data, err := json.Marshal(pingMsg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}
	data = append(data, '\n')
	conn.readBuf.Write(data)

	// Обрабатываем соединение в отдельной горутине
	done := make(chan bool)
	go func() {
		node.handleConnection(conn, "192.168.1.1:8081")
		done <- true
	}()

	// Ждем завершения обработки или таймаут
	select {
	case <-done:
		// Проверяем, что ответ был отправлен
		if conn.writeBuf.Len() == 0 {
			t.Errorf("Expected response to be written to connection, but buffer is empty")
		}

		// Декодируем ответ
		var response NetworkMessage
		if err := json.NewDecoder(conn.writeBuf).Decode(&response); err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}

		// Проверяем тип ответа
		if response.Type != "pong" {
			t.Errorf("Expected response type to be pong, got %s", response.Type)
		}

	case <-time.After(2 * time.Second):
		t.Errorf("Timeout waiting for handleConnection to complete")
	}
}

func TestBroadcastMessage(t *testing.T) {
	bc := blockchain.NewBlockchain()
	node := NewNode("127.0.0.1:8080", bc, nil)

	// Создаем мок-соединения для пиров
	conn1 := newMockConn()
	conn2 := newMockConn()

	// Добавляем пиры вручную
	node.mutex.Lock()
	node.Peers["192.168.1.1:8081"] = conn1
	node.Peers["192.168.1.2:8082"] = conn2
	node.mutex.Unlock()

	// Создаем сообщение для отправки
	block := blockchain.Block{
		Index:     1,
		Timestamp: time.Now().Unix(),
		Messages:  []blockchain.Message{{Sender: "Alice", Recipient: "Bob", Content: "Test", Timestamp: time.Now().Unix()}},
		PrevHash:  "0000000000000000000000000000000000000000000000000000000000000000",
		Hash:      "0000abcdef0123456789abcdef0123456789abcdef0123456789abcdef012345",
		Nonce:     42,
	}

	// Отправляем сообщение
	node.BroadcastBlock(block)

	// Проверяем, что сообщение было отправлено обоим пирам
	for i, conn := range []*mockConn{conn1, conn2} {
		if conn.writeBuf.Len() == 0 {
			t.Errorf("Expected message to be written to connection %d, but buffer is empty", i+1)
		}

		// Декодируем отправленное сообщение
		var sentMessage NetworkMessage
		if err := json.NewDecoder(conn.writeBuf).Decode(&sentMessage); err != nil {
			t.Errorf("Failed to decode sent message for connection %d: %v", i+1, err)
		}

		// Проверяем тип сообщения
		if sentMessage.Type != "new_block" {
			t.Errorf("Expected message type to be new_block, got %s for connection %d", sentMessage.Type, i+1)
		}
	}
}

func TestRequestChain(t *testing.T) {
	bc := blockchain.NewBlockchain()
	node := NewNode("127.0.0.1:8080", bc, nil)

	// Создаем мок-соединение
	conn := newMockConn()

	// Добавляем пир вручную
	node.mutex.Lock()
	node.Peers["192.168.1.1:8081"] = conn
	node.mutex.Unlock()

	// Запрашиваем цепочку
	node.requestChain("192.168.1.1:8081")

	// Проверяем, что запрос был отправлен
	if conn.writeBuf.Len() == 0 {
		t.Errorf("Expected request to be written to connection, but buffer is empty")
	}

	// Декодируем отправленный запрос
	var request NetworkMessage
	if err := json.NewDecoder(conn.writeBuf).Decode(&request); err != nil {
		t.Errorf("Failed to decode request: %v", err)
	}

	// Проверяем тип запроса
	if request.Type != "chain_request" {
		t.Errorf("Expected request type to be chain_request, got %s", request.Type)
	}
}

func TestSendChain(t *testing.T) {
	bc := blockchain.NewBlockchain()
	node := NewNode("127.0.0.1:8080", bc, nil)

	// Создаем мок-соединение
	conn := newMockConn()

	// Отправляем цепочку
	node.sendChain(conn)

	// Проверяем, что ответ был отправлен
	if conn.writeBuf.Len() == 0 {
		t.Errorf("Expected response to be written to connection, but buffer is empty")
	}

	// Декодируем отправленный ответ
	var response NetworkMessage
	if err := json.NewDecoder(conn.writeBuf).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	// Проверяем тип ответа
	if response.Type != "chain_response" {
		t.Errorf("Expected response type to be chain_response, got %s", response.Type)
	}
}

func TestHandleChainResponse(t *testing.T) {
	bc := blockchain.NewBlockchain()
	node := NewNode("127.0.0.1:8080", bc, nil)

	// Создаем более длинную цепочку для ответа
	longerChain := make([]blockchain.Block, 3)
	longerChain[0] = bc.Chain[0] // Генезис-блок

	// Добавляем два новых блока (майним их, чтобы они удовлетворяли PoW)
	longerChain[1] = blockchain.SimpleMineBlock(blockchain.Block{
		Index:     1,
		Timestamp: time.Now().Unix(),
		Messages:  []blockchain.Message{{Sender: "Alice", Recipient: "Bob", Content: "Test1", Timestamp: time.Now().Unix()}},
		PrevHash:  longerChain[0].Hash,
	}, bc.Difficulty)

	longerChain[2] = blockchain.SimpleMineBlock(blockchain.Block{
		Index:     2,
		Timestamp: time.Now().Unix(),
		Messages:  []blockchain.Message{{Sender: "Bob", Recipient: "Alice", Content: "Test2", Timestamp: time.Now().Unix()}},
		PrevHash:  longerChain[1].Hash,
	}, bc.Difficulty)

	// Обрабатываем ответ
	node.handleChainResponse(longerChain)

	// Проверяем, что цепочка была обновлена
	if len(node.Blockchain.Chain) != 3 {
		t.Errorf("Expected blockchain to have 3 blocks after handling chain response, got %d", len(node.Blockchain.Chain))
	}

	// Проверяем, что последний блок соответствует ожидаемому
	lastBlock := node.Blockchain.GetLastBlock()
	if lastBlock.Index != 2 {
		t.Errorf("Expected last block index to be 2, got %d", lastBlock.Index)
	}

	if lastBlock.Hash != longerChain[2].Hash {
		t.Errorf("Expected last block hash to be %s, got %s", longerChain[2].Hash, lastBlock.Hash)
	}
}

func TestHandleNewBlock(t *testing.T) {
	bc := blockchain.NewBlockchain()
	node := NewNode("127.0.0.1:8080", bc, nil)

	// Создаем новый блок (майним его, чтобы он удовлетворял PoW)
	newBlock := blockchain.SimpleMineBlock(blockchain.Block{
		Index:     1,
		Timestamp: time.Now().Unix(),
		Messages:  []blockchain.Message{{Sender: "Alice", Recipient: "Bob", Content: "Test", Timestamp: time.Now().Unix()}},
		PrevHash:  bc.Chain[0].Hash,
	}, bc.Difficulty)

	// Обрабатываем новый блок
	node.handleNewBlock(newBlock)

	// Проверяем, что блок был добавлен в цепочку
	if len(bc.Chain) != 2 {
		t.Errorf("Expected blockchain to have 2 blocks after handling new block, got %d", len(bc.Chain))
	}

	// Проверяем, что добавленный блок соответствует ожидаемому
	lastBlock := bc.GetLastBlock()
	if lastBlock.Index != 1 {
		t.Errorf("Expected last block index to be 1, got %d", lastBlock.Index)
	}

	if lastBlock.Hash != newBlock.Hash {
		t.Errorf("Expected last block hash to be %s, got %s", newBlock.Hash, lastBlock.Hash)
	}

	// Создаем невалидный блок (неправильный PrevHash)
	invalidBlock := blockchain.SimpleMineBlock(blockchain.Block{
		Index:     2,
		Timestamp: time.Now().Unix(),
		Messages:  []blockchain.Message{{Sender: "Bob", Recipient: "Alice", Content: "Test2", Timestamp: time.Now().Unix()}},
		PrevHash:  "invalid_prev_hash",
	}, bc.Difficulty)

	// Обрабатываем невалидный блок
	node.handleNewBlock(invalidBlock)

	// Проверяем, что невалидный блок не был добавлен
	if len(bc.Chain) != 2 {
		t.Errorf("Expected blockchain to still have 2 blocks after handling invalid block, got %d", len(bc.Chain))
	}
}

func TestSendPing(t *testing.T) {
	bc := blockchain.NewBlockchain()
	node := NewNode("127.0.0.1:8080", bc, nil)

	// Создаем мок-соединение
	conn := newMockConn()

	// Добавляем пир вручную
	node.mutex.Lock()
	node.Peers["192.168.1.1:8081"] = conn
	node.mutex.Unlock()

	// Отправляем пинг
	node.sendPing("192.168.1.1:8081")

	// Проверяем, что пинг был отправлен
	if conn.writeBuf.Len() == 0 {
		t.Errorf("Expected ping to be written to connection, but buffer is empty")
	}

	// Декодируем отправленный пинг
	var ping NetworkMessage
	if err := json.NewDecoder(conn.writeBuf).Decode(&ping); err != nil {
		t.Errorf("Failed to decode ping: %v", err)
	}

	// Проверяем тип пинга
	if ping.Type != "ping" {
		t.Errorf("Expected ping type to be ping, got %s", ping.Type)
	}
}

func TestSendPong(t *testing.T) {
	bc := blockchain.NewBlockchain()
	node := NewNode("127.0.0.1:8080", bc, nil)

	// Создаем мок-соединение
	conn := newMockConn()

	// Отправляем понг
	node.sendPong(conn)

	// Проверяем, что понг был отправлен
	if conn.writeBuf.Len() == 0 {
		t.Errorf("Expected pong to be written to connection, but buffer is empty")
	}

	// Декодируем отправленный понг
	var pong NetworkMessage
	if err := json.NewDecoder(conn.writeBuf).Decode(&pong); err != nil {
		t.Errorf("Failed to decode pong: %v", err)
	}

	// Проверяем тип понга
	if pong.Type != "pong" {
		t.Errorf("Expected pong type to be pong, got %s", pong.Type)
	}
}
