package tests

import (
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/freeeakn/AetherWave/pkg/blockchain"
	"github.com/freeeakn/AetherWave/pkg/crypto"
	"github.com/freeeakn/AetherWave/pkg/network"
)

// labeledPipeConn wraps a net.Conn to return distinct addresses with unique ports
type labeledPipeConn struct {
	net.Conn
	localAddr  net.Addr
	remoteAddr net.Addr
}

func (c *labeledPipeConn) LocalAddr() net.Addr  { return c.localAddr }
func (c *labeledPipeConn) RemoteAddr() net.Addr { return c.remoteAddr }

// pipeListener implements net.Listener using net.Pipe() for in-memory connections
type pipeListener struct {
	connCh chan net.Conn
	addr   net.Addr
	closed bool
	mu     sync.Mutex
	index  int
}

func newPipeListener(addr string) *pipeListener {
	host, portStr, _ := net.SplitHostPort(addr)
	port, _ := strconv.Atoi(portStr)
	if host == "" {
		host = "127.0.0.1"
	}
	return &pipeListener{
		connCh: make(chan net.Conn, 100),
		addr:   &net.TCPAddr{IP: net.ParseIP(host), Port: port},
	}
}

func (l *pipeListener) Accept() (net.Conn, error) {
	conn, ok := <-l.connCh
	if !ok {
		return nil, fmt.Errorf("listener closed")
	}
	return conn, nil
}

func (l *pipeListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.closed {
		l.closed = true
		close(l.connCh)
	}
	return nil
}

func (l *pipeListener) Addr() net.Addr { return l.addr }

// setupTestNodes creates test nodes connected via net.Pipe().
// Each node gets a pipeListener; ConnectToPeer creates a pipe pair
// and routes one end to the remote listener's Accept chann el.
func setupTestNodes(t *testing.T, count int) []*network.Node {
	t.Helper()

	originalListenFn := network.ListenFn
	originalDialPeer := network.DialPeer
	t.Cleanup(func() {
		network.ListenFn = originalListenFn
		network.DialPeer = originalDialPeer
	})

	listenerMap := make(map[string]*pipeListener)
	nodeAddrs := make([]string, count)
	for i := 0; i < count; i++ {
		addr := fmt.Sprintf("127.0.0.1:%d", 8000+i)
		nodeAddrs[i] = addr
		listenerMap[addr] = newPipeListener(addr)
	}

	network.ListenFn = func(_, address string) (net.Listener, error) {
		l, ok := listenerMap[address]
		if !ok {
			return nil, fmt.Errorf("no listener for %s", address)
		}
		return l, nil
	}

	nodes := make([]*network.Node, count)
	for i := 0; i < count; i++ {
		bc := blockchain.NewBlockchain()
		address := nodeAddrs[i]

		var bootstrapPeers []string
		for j := 0; j < i; j++ {
			bootstrapPeers = append(bootstrapPeers, nodeAddrs[j])
		}

		nodes[i] = network.NewNode(address, bc, bootstrapPeers)
	}

	var pipeCounter int
	var pipeMu sync.Mutex
	network.DialPeer = func(address string) (net.Conn, error) {
		l, ok := listenerMap[address]
		if !ok {
			return nil, fmt.Errorf("unknown peer: %s", address)
		}
		c1, c2 := net.Pipe()
		pipeMu.Lock()
		pipeCounter++
		portA := 50000 + pipeCounter*2
		portB := portA + 1
		pipeMu.Unlock()

		// Use unique port numbers so each pipe connection has distinct address strings
		wc1 := &labeledPipeConn{
			Conn:       c1,
			localAddr:  &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: portA},
			remoteAddr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: portB},
		}
		wc2 := &labeledPipeConn{
			Conn:       c2,
			localAddr:  &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: portB},
			remoteAddr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: portA},
		}
		l.connCh <- wc2
		return wc1, nil
	}

	for _, node := range nodes {
		go func(n *network.Node) {
			if err := n.Start(); err != nil {
				t.Errorf("Failed to start node %s: %v", n.Address, err)
			}
		}(node)
	}

	time.Sleep(200 * time.Millisecond)

	return nodes
}

// TestNodeCommunication проверяет обмен сообщениями между узлами
func TestNodeCommunication(t *testing.T) {
	nodes := setupTestNodes(t, 2)
	defer func() {
		for _, n := range nodes {
			n.Stop()
		}
	}()

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	err = nodes[0].Blockchain.AddMessage("Alice", "Bob", "Hello from node1!", key)
	if err != nil {
		t.Fatalf("Failed to add message to node1: %v", err)
	}

	lastBlock := nodes[0].Blockchain.GetLastBlock()

	nodes[0].BroadcastBlock(lastBlock)
	time.Sleep(500 * time.Millisecond)

	lastBlock2 := nodes[1].Blockchain.GetLastBlock()
	if lastBlock2.Index != lastBlock.Index {
		t.Errorf("Expected node2 to have block index %d, got %d", lastBlock.Index, lastBlock2.Index)
	}
	if lastBlock2.Hash != lastBlock.Hash {
		t.Errorf("Expected node2 to have hash %s, got %s", lastBlock.Hash, lastBlock2.Hash)
	}

	messages := nodes[1].Blockchain.ReadMessages("Bob", key)
	if len(messages) != 1 {
		t.Errorf("Expected 1 message from node2, got %d", len(messages))
	}
	if len(messages) > 0 && !contains(messages[0], "Hello from node1!") {
		t.Errorf("Expected message to contain 'Hello from node1!', got %s", messages[0])
	}
}

// TestBlockchainSynchronization проверяет синхронизацию блокчейна между узлами
func TestBlockchainSynchronization(t *testing.T) {
	nodes := setupTestNodes(t, 3)
	defer func() {
		for _, n := range nodes {
			n.Stop()
		}
	}()

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	for i := 0; i < 3; i++ {
		err = nodes[0].Blockchain.AddMessage("Alice", "Bob", fmt.Sprintf("Message %d", i), key)
		if err != nil {
			t.Fatalf("Failed to add message %d: %v", i, err)
		}
	}

	lastBlock := nodes[0].Blockchain.GetLastBlock()
	nodes[0].BroadcastBlock(lastBlock)

	time.Sleep(500 * time.Millisecond)

	if nodes[1].Blockchain.GetLastBlock().Index != nodes[0].Blockchain.GetLastBlock().Index {
		t.Errorf("Expected node2 chain length %d, got %d",
			nodes[0].Blockchain.GetLastBlock().Index, nodes[1].Blockchain.GetLastBlock().Index)
	}
	if nodes[2].Blockchain.GetLastBlock().Hash != nodes[0].Blockchain.GetLastBlock().Hash {
		t.Errorf("Expected node3 to have same last block hash as node1")
	}

	messages := nodes[2].Blockchain.ReadMessages("Bob", key)
	if len(messages) != 3 {
		t.Errorf("Expected 3 messages from node3, got %d", len(messages))
	}
}

// TestEndToEndMessaging проверяет полный цикл отправки и получения сообщений
func TestEndToEndMessaging(t *testing.T) {
	logger, cleanup := SetupTestLogging(t)
	defer cleanup()

	logger.Log("Starting end-to-end messaging test")

	bc := blockchain.NewBlockchain()
	logger.Log("Created blockchain instance")

	key, err := crypto.GenerateKey()
	AssertNoError(t, err, "Failed to generate key")
	logger.Log("Generated encryption key")

	messages := []struct {
		sender    string
		recipient string
		content   string
	}{
		{"Alice", "Bob", "Hello, Bob!"},
		{"Bob", "Alice", "Hi Alice, how are you?"},
		{"Alice", "Bob", "I'm fine, thanks!"},
		{"Charlie", "Alice", "Hello Alice, it's Charlie!"},
		{"Alice", "Charlie", "Hi Charlie!"},
	}

	logger.Log("Adding %d messages to blockchain", len(messages))
	for i, msg := range messages {
		err = bc.AddMessage(msg.sender, msg.recipient, msg.content, key)
		AssertNoError(t, err, fmt.Sprintf("Failed to add message %d", i))
		logger.Log("Added message %d: %s -> %s", i, msg.sender, msg.recipient)
	}

	expectedBlockCount := len(messages) + 1
	actualBlockCount := len(bc.GetChain())
	AssertEqual(t, expectedBlockCount, actualBlockCount, "Incorrect blockchain length")
	logger.Log("Blockchain contains %d blocks (including genesis block)", actualBlockCount)

	bobMessages := bc.ReadMessages("Bob", key)
	AssertEqual(t, 2, len(bobMessages), "Incorrect number of messages for Bob")
	logger.Log("Bob has %d messages", len(bobMessages))

	aliceMessages := bc.ReadMessages("Alice", key)
	AssertEqual(t, 2, len(aliceMessages), "Incorrect number of messages for Alice")
	logger.Log("Alice has %d messages", len(aliceMessages))

	charlieMessages := bc.ReadMessages("Charlie", key)
	AssertEqual(t, 1, len(charlieMessages), "Incorrect number of messages for Charlie")
	logger.Log("Charlie has %d messages", len(charlieMessages))

	logger.Log("Verifying message contents")
	for i, msg := range bobMessages {
		if !contains(msg, "Hello, Bob!") && !contains(msg, "I'm fine, thanks!") {
			t.Errorf("Bob's message %d doesn't contain expected content: %s", i, msg)
			logger.Log("ERROR: Bob's message %d has unexpected content: %s", i, msg)
		} else {
			logger.Log("Bob's message %d has expected content", i)
		}
	}

	for i, msg := range aliceMessages {
		if !contains(msg, "Hi Alice, how are you?") && !contains(msg, "Hello Alice, it's Charlie!") {
			t.Errorf("Alice's message %d doesn't contain expected content: %s", i, msg)
			logger.Log("ERROR: Alice's message %d has unexpected content: %s", i, msg)
		} else {
			logger.Log("Alice's message %d has expected content", i)
		}
	}

	for i, msg := range charlieMessages {
		if !contains(msg, "Hi Charlie!") {
			t.Errorf("Charlie's message %d doesn't contain expected content: %s", i, msg)
			logger.Log("ERROR: Charlie's message %d has unexpected content: %s", i, msg)
		} else {
			logger.Log("Charlie's message %d has expected content", i)
		}
	}

	logger.Log("Verifying message encryption")
	for i, block := range bc.GetChain() {
		for j, msg := range block.Messages {
			if msg.Content == "" {
				logger.Log("Block %d, message %d: empty content (likely genesis block)", i, j)
				continue
			}

			_, err := hex.DecodeString(msg.Content)
			if err != nil {
				t.Errorf("Message content is not a valid hex string: %v", err)
				logger.Log("ERROR: Block %d, message %d: content is not a valid hex string", i, j)
			} else {
				logger.Log("Block %d, message %d: content is a valid hex string", i, j)
			}

			for k, origMsg := range messages {
				if msg.Sender == origMsg.sender && msg.Recipient == origMsg.recipient && msg.Content == origMsg.content {
					t.Errorf("Message content is not encrypted: %s", msg.Content)
					logger.Log("ERROR: Block %d, message %d (original message %d): content is not encrypted", i, j, k)
				}
			}
		}
	}

	logger.Log("End-to-end messaging test completed successfully")
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
