package blockchain

import (
	"strings"
	"testing"
	"time"
)

func TestNewBlockchain(t *testing.T) {
	bc := NewBlockchain()

	if bc == nil {
		t.Fatal("NewBlockchain() returned nil")
	}

	if len(bc.Chain) != 1 {
		t.Errorf("Expected blockchain to have 1 genesis block, got %d", len(bc.Chain))
	}

	genesisBlock := bc.Chain[0]
	if genesisBlock.Index != 0 {
		t.Errorf("Expected genesis block index to be 0, got %d", genesisBlock.Index)
	}

	if genesisBlock.PrevHash != "0" {
		t.Errorf("Expected genesis block PrevHash to be '0', got %s", genesisBlock.PrevHash)
	}

	if genesisBlock.Timestamp != GenesisTimestamp {
		t.Errorf("Expected genesis block timestamp to be %d, got %d", GenesisTimestamp, genesisBlock.Timestamp)
	}

	if len(genesisBlock.Messages) != 0 {
		t.Errorf("Expected genesis block to have 0 messages, got %d", len(genesisBlock.Messages))
	}

	if genesisBlock.Hash != SimpleCalculateHash(genesisBlock) {
		t.Errorf("Expected genesis block hash to match calculated hash")
	}
}

func TestSimpleCalculateHash(t *testing.T) {
	block := Block{
		Index:     1,
		Timestamp: 1677654322,
		Messages:  []Message{{Sender: "Alice", Recipient: "Bob", Content: "Test", Timestamp: 1677654322}},
		PrevHash:  "0000abcdef0123456789abcdef0123456789abcdef0123456789abcdef012345",
		Nonce:     42,
	}

	hash := SimpleCalculateHash(block)

	if len(hash) != 64 {
		t.Errorf("Expected hash length to be 64 characters, got %d", len(hash))
	}

	// Проверяем, что хеш изменяется при изменении данных блока
	block.Nonce = 43
	newHash := SimpleCalculateHash(block)

	if hash == newHash {
		t.Errorf("Expected hash to change when block data changes")
	}
}

func TestSimpleMineBlock(t *testing.T) {
	block := Block{
		Index:     1,
		Timestamp: time.Now().Unix(),
		Messages:  []Message{{Sender: "Alice", Recipient: "Bob", Content: "Test", Timestamp: time.Now().Unix()}},
		PrevHash:  "0000abcdef0123456789abcdef0123456789abcdef0123456789abcdef012345",
	}

	difficulty := 2
	minedBlock := SimpleMineBlock(block, difficulty)

	// Проверяем, что хеш начинается с нужного количества нулей
	if !strings.HasPrefix(minedBlock.Hash, strings.Repeat("0", difficulty)) {
		t.Errorf("Expected mined block hash to start with %d zeros, got %s", difficulty, minedBlock.Hash)
	}

	// Проверяем, что хеш соответствует блоку
	if minedBlock.Hash != SimpleCalculateHash(minedBlock) {
		t.Errorf("Expected mined block hash to match calculated hash")
	}
}

func TestAddMessage(t *testing.T) {
	bc := NewBlockchain()
	key := make([]byte, 32) // Нулевой ключ для тестирования

	err := bc.AddMessage("Alice", "Bob", "Hello, Bob!", key)
	if err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}

	if len(bc.Chain) != 2 {
		t.Errorf("Expected blockchain to have 2 blocks after adding message, got %d", len(bc.Chain))
	}

	lastBlock := bc.Chain[len(bc.Chain)-1]
	if lastBlock.Index != 1 {
		t.Errorf("Expected new block index to be 1, got %d", lastBlock.Index)
	}

	if len(lastBlock.Messages) != 1 {
		t.Errorf("Expected new block to have 1 message, got %d", len(lastBlock.Messages))
	}

	if lastBlock.PrevHash != bc.Chain[0].Hash {
		t.Errorf("Expected new block PrevHash to match genesis block hash")
	}

	// Проверяем хеш блока
	if !strings.HasPrefix(lastBlock.Hash, strings.Repeat("0", bc.Difficulty)) {
		t.Errorf("Expected block hash to start with %d zeros, got %s", bc.Difficulty, lastBlock.Hash)
	}
}

func TestReadMessages(t *testing.T) {
	bc := NewBlockchain()
	key := make([]byte, 32) // Нулевой ключ для тестирования

	// Добавляем несколько сообщений с разными получателями
	if err := bc.AddMessage("Alice", "Bob", "Hello, Bob!", key); err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}
	if err := bc.AddMessage("Charlie", "Bob", "Hey Bob, how are you?", key); err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}
	if err := bc.AddMessage("Bob", "Alice", "Hi Alice!", key); err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}
	if err := bc.AddMessage("Dave", "Eve", "Hello Eve!", key); err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}

	// Читаем сообщения для Bob
	bobMessages := bc.ReadMessages("Bob", key)
	if len(bobMessages) != 2 {
		t.Errorf("Expected Bob to have 2 messages, got %d", len(bobMessages))
	}

	// Читаем сообщения для Alice
	aliceMessages := bc.ReadMessages("Alice", key)
	if len(aliceMessages) != 1 {
		t.Errorf("Expected Alice to have 1 message, got %d", len(aliceMessages))
	}

	// Читаем сообщения для Eve
	eveMessages := bc.ReadMessages("Eve", key)
	if len(eveMessages) != 1 {
		t.Errorf("Expected Eve to have 1 message, got %d", len(eveMessages))
	}

	// Читаем сообщения для несуществующего получателя
	nonexistentMessages := bc.ReadMessages("Nonexistent", key)
	if len(nonexistentMessages) != 0 {
		t.Errorf("Expected Nonexistent to have 0 messages, got %d", len(nonexistentMessages))
	}

	// Проверяем содержимое сообщений
	for _, msg := range bobMessages {
		if !strings.Contains(msg, "From Alice") && !strings.Contains(msg, "From Charlie") {
			t.Errorf("Expected message to be from Alice or Charlie, got: %s", msg)
		}
	}

	for _, msg := range aliceMessages {
		if !strings.Contains(msg, "From Bob") {
			t.Errorf("Expected message to be from Bob, got: %s", msg)
		}
	}

	for _, msg := range eveMessages {
		if !strings.Contains(msg, "From Dave") {
			t.Errorf("Expected message to be from Dave, got: %s", msg)
		}
	}
}

func TestVerifyChain(t *testing.T) {
	bc := NewBlockchain()
	key := make([]byte, 32) // Нулевой ключ для тестирования

	// Добавляем несколько сообщений для создания блоков
	if err := bc.AddMessage("Alice", "Bob", "Hello, Bob!", key); err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}
	if err := bc.AddMessage("Charlie", "Bob", "Hey Bob, how are you?", key); err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}

	// Проверяем, что цепочка валидна
	if !bc.VerifyChain() {
		t.Errorf("Expected chain to be valid")
	}

	// Имитируем взлом, изменяя сообщение в первом блоке без пересчёта хеша
	tamperedChain := bc.GetChain()
	tamperedChain[1].Messages[0].Content = "Tampered content"
	// НЕ пересчитываем хеш — хеш блока теперь не соответствует содержимому

	// Заменяем оригинальную цепочку взломанной
	bc.Chain = tamperedChain

	// Проверяем, что цепочка теперь невалидна
	if bc.VerifyChain() {
		t.Errorf("Expected tampered chain to be invalid")
	}
}

func TestGetLastBlock(t *testing.T) {
	bc := NewBlockchain()
	key := make([]byte, 32) // Нулевой ключ для тестирования

	// Добавляем сообщение для создания нового блока
	bc.AddMessage("Alice", "Bob", "Hello, Bob!", key)

	lastBlock := bc.GetLastBlock()
	if lastBlock.Index != 1 {
		t.Errorf("Expected last block index to be 1, got %d", lastBlock.Index)
	}

	if len(lastBlock.Messages) != 1 {
		t.Errorf("Expected last block to have 1 message, got %d", len(lastBlock.Messages))
	}

	// Проверяем, что GetLastBlock возвращает правильный блок
	if lastBlock.Hash != bc.Chain[len(bc.Chain)-1].Hash {
		t.Errorf("Expected GetLastBlock to return the same block as the last in the chain")
	}
}

func TestGetChain(t *testing.T) {
	bc := NewBlockchain()
	key := make([]byte, 32) // Нулевой ключ для тестирования

	// Добавляем сообщение для создания нового блока
	bc.AddMessage("Alice", "Bob", "Hello, Bob!", key)

	chain := bc.GetChain()
	if len(chain) != 2 {
		t.Errorf("Expected chain length to be 2, got %d", len(chain))
	}

	// Проверяем, что GetChain возвращает копию цепочки
	chain[0].Index = 999
	if bc.Chain[0].Index == 999 {
		t.Errorf("Expected GetChain to return a copy of the chain, not a reference")
	}
}

func TestUpdateChain(t *testing.T) {
	bc1 := NewBlockchain()
	bc2 := NewBlockchain()

	key := make([]byte, 32) // Нулевой ключ для тестирования

	// Добавляем блоки в bc2, делая его длиннее bc1
	if err := bc2.AddMessage("Alice", "Bob", "Hello from bc2!", key); err != nil {
		t.Fatalf("Failed to add message to bc2: %v", err)
	}
	if err := bc2.AddMessage("Charlie", "Dave", "Another message in bc2", key); err != nil {
		t.Fatalf("Failed to add message to bc2: %v", err)
	}

	// Получаем копию цепочки bc2
	longerChain := bc2.GetChain()

	// Обновляем bc1 с помощью цепочки bc2
	updated := bc1.UpdateChain(longerChain)

	if !updated {
		t.Errorf("Expected UpdateChain to return true for a longer valid chain")
	}

	if len(bc1.Chain) != len(longerChain) {
		t.Errorf("Expected bc1 chain length to be %d after update, got %d", len(longerChain), len(bc1.Chain))
	}

	// Попытка обновить с короткой цепочкой должна не удаться
	shortChain := bc1.GetChain()[:1] // Только genesis блок
	updated = bc2.UpdateChain(shortChain)

	if updated {
		t.Errorf("Expected UpdateChain to return false for a shorter chain")
	}

	// Создаём более длинную недействительную цепочку (длиннее bc1, чтобы UpdateChain проверял хеши)
	longerInvalidChain := bc2.GetChain()
	badBlock := Block{
		Index:     len(longerInvalidChain),
		Timestamp: time.Now().Unix(),
		PrevHash:  "invalid_prev_hash",
	}
	badBlock.Hash = SimpleCalculateHash(badBlock)
	longerInvalidChain = append(longerInvalidChain, badBlock)

	updated = bc1.UpdateChain(longerInvalidChain)

	if updated {
		t.Errorf("Expected UpdateChain to return false for an invalid chain")
	}
}
