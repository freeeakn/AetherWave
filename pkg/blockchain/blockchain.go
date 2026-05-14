package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/freeeakn/AetherWave/pkg/crypto"
)

const DefaultMiningTimeout = 60 * time.Second

// Message представляет сообщение в блокчейне
type Message struct {
	Sender    string
	Recipient string
	Content   string
	Timestamp int64
}

// Block представляет блок в блокчейне
type Block struct {
	Index     int
	Timestamp int64
	Messages  []Message
	PrevHash  string
	Hash      string
	Nonce     int
}

// Blockchain представляет цепочку блоков
type Blockchain struct {
	Chain      []Block
	Difficulty int
	mutex      sync.RWMutex
}

// GenesisTimestamp - фиксированная временная метка для генезис-блока
const GenesisTimestamp = 1677654321

// NewBlockchain создает новый экземпляр блокчейна с генезис-блоком
func NewBlockchain() *Blockchain {
	genesisBlock := Block{
		Index:     0,
		Timestamp: GenesisTimestamp,
		Messages:  []Message{},
		PrevHash:  "0",
	}
	genesisBlock.Hash = SimpleCalculateHash(genesisBlock)
	return &Blockchain{
		Chain:      []Block{genesisBlock},
		Difficulty: 4,
	}
}

// CalculateHash вычисляет хеш блока (единая реализация для всего проекта)
func CalculateHash(block Block) string {
	data := fmt.Sprintf("%d%d%s%d%s",
		block.Index,
		block.Timestamp,
		block.PrevHash,
		block.Nonce,
		serializeMessages(block.Messages))
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

// SimpleCalculateHash сохранён для обратной совместимости, делегирует в CalculateHash
func SimpleCalculateHash(block Block) string {
	return CalculateHash(block)
}

// AddMessage добавляет новое сообщение в блокчейн
func (bc *Blockchain) AddMessage(sender, recipient, content string, key []byte) error {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	encryptedContent, err := crypto.EncryptMessage(content, key)
	if err != nil {
		return fmt.Errorf("encryption error: %v", err)
	}

	newMessage := Message{
		Sender:    sender,
		Recipient: recipient,
		Content:   hex.EncodeToString(encryptedContent),
		Timestamp: time.Now().Unix(),
	}

	lastBlock := bc.Chain[len(bc.Chain)-1]
	newBlock := Block{
		Index:     lastBlock.Index + 1,
		Timestamp: time.Now().Unix(),
		Messages:  []Message{newMessage},
		PrevHash:  lastBlock.Hash,
	}

	// Используем простой майнинг блока
	minedBlock := SimpleMineBlock(newBlock, bc.Difficulty)
	bc.Chain = append(bc.Chain, minedBlock)
	return nil
}

// ReadMessages читает сообщения для указанного получателя
func (bc *Blockchain) ReadMessages(recipient string, key []byte) []string {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	var messages []string
	for _, block := range bc.Chain {
		for _, msg := range block.Messages {
			if msg.Recipient == recipient {
				encryptedBytes, err := hex.DecodeString(msg.Content)
				if err != nil {
					continue
				}
				decrypted, err := crypto.DecryptMessage(encryptedBytes, key)
				if err != nil {
					continue
				}
				messages = append(messages, fmt.Sprintf("From %s: %s", msg.Sender, decrypted))
			}
		}
	}
	return messages
}

// meetsDifficulty проверяет, удовлетворяет ли хеш заданной сложности
func meetsDifficulty(hash string, difficulty int) bool {
	if difficulty <= 0 {
		return true
	}
	if len(hash) < difficulty {
		return false
	}
	for i := 0; i < difficulty; i++ {
		if hash[i] != '0' {
			return false
		}
	}
	return true
}

// SimpleMineBlock выполняет простой майнинг блока с заданной сложностью
func SimpleMineBlock(block Block, difficulty int) Block {
	deadline := time.Now().Add(DefaultMiningTimeout)
	hitDeadline := false
	for {
		if !hitDeadline && time.Now().After(deadline) {
			hitDeadline = true
			fmt.Println("Warning: mining timeout reached, continuing until valid hash found")
		}
		hash := SimpleCalculateHash(block)
		if meetsDifficulty(hash, difficulty) {
			block.Hash = hash
			return block
		}
		block.Nonce++
	}
}

// SaveToFile сохраняет блокчейн в JSON-файл
func (bc *Blockchain) SaveToFile(path string) error {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	data, err := json.Marshal(bc.Chain)
	if err != nil {
		return fmt.Errorf("failed to marshal chain: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write chain file: %v", err)
	}
	return nil
}

// LoadFromFile загружает блокчейн из JSON-файла. Цепочка загружается только если она длиннее текущей и валидна.
func (bc *Blockchain) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read chain file: %v", err)
	}

	var loadedChain []Block
	if err := json.Unmarshal(data, &loadedChain); err != nil {
		return fmt.Errorf("failed to unmarshal chain: %v", err)
	}

	if len(loadedChain) == 0 {
		return nil
	}

	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	if len(loadedChain) <= len(bc.Chain) {
		return nil
	}

	valid := true
	expectedPrevHash := "0"
	for _, block := range loadedChain {
		if block.PrevHash != expectedPrevHash {
			valid = false
			break
		}
		if block.Hash != SimpleCalculateHash(block) {
			valid = false
			break
		}
		if block.Index > 0 && !meetsDifficulty(block.Hash, bc.Difficulty) {
			valid = false
			break
		}
		expectedPrevHash = block.Hash
	}

	if valid {
		bc.Chain = loadedChain
		fmt.Printf("Loaded chain from %s (%d blocks)\n", path, len(loadedChain))
	}

	return nil
}

// VerifyChain проверяет целостность блокчейна
func (bc *Blockchain) VerifyChain() bool {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	for i := 1; i < len(bc.Chain); i++ {
		current := bc.Chain[i]
		previous := bc.Chain[i-1]
		if current.Hash != SimpleCalculateHash(current) || current.PrevHash != previous.Hash {
			return false
		}
		if !meetsDifficulty(current.Hash, bc.Difficulty) {
			return false
		}
	}
	return true
}

// GetLastBlock возвращает последний блок в цепочке
func (bc *Blockchain) GetLastBlock() Block {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()
	return bc.Chain[len(bc.Chain)-1]
}

// GetChain возвращает глубокую копию всей цепочки блоков
func (bc *Blockchain) GetChain() []Block {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	chainCopy := make([]Block, len(bc.Chain))
	for i, b := range bc.Chain {
		chainCopy[i] = Block{
			Index:     b.Index,
			Timestamp: b.Timestamp,
			Messages:  append([]Message{}, b.Messages...),
			PrevHash:  b.PrevHash,
			Hash:      b.Hash,
			Nonce:     b.Nonce,
		}
	}
	return chainCopy
}

// UpdateChain обновляет цепочку блоков, если новая цепочка длиннее и валидна
func (bc *Blockchain) UpdateChain(newChain []Block) bool {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	if len(newChain) <= len(bc.Chain) {
		return false
	}

	valid := true
	expectedPrevHash := "0"
	for _, block := range newChain {
		if block.PrevHash != expectedPrevHash {
			valid = false
			break
		}
		if block.Hash != SimpleCalculateHash(block) {
			valid = false
			break
		}
		if block.Index > 0 && !meetsDifficulty(block.Hash, bc.Difficulty) {
			valid = false
			break
		}
		expectedPrevHash = block.Hash
	}

	if valid {
		bc.Chain = newChain
		return true
	}
	return false
}
