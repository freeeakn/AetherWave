package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/freeeakn/AetherWave/pkg/blockchain"
	"github.com/freeeakn/AetherWave/pkg/crypto"
	"github.com/freeeakn/AetherWave/pkg/network"
)

// Build-time variables set via -ldflags in goreleaser
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	address := flag.String("address", ":3000", "Node P2P address (e.g., :3000 or 192.168.1.100:3000)")
	apiAddress := flag.String("api-address", ":8080", "HTTP API address (e.g., :8080)")
	peer := flag.String("peer", "", "Initial peer address to connect to")
	name := flag.String("name", "User", "Your username")
	keyStr := flag.String("key", "", "Shared encryption key (hex-encoded, 32 bytes); if empty, one will be generated")
	chainFile := flag.String("chain-file", "", "Path to persist blockchain (JSON); empty means no persistence")
	enableDiscovery := flag.Bool("discovery", false, "Enable automatic node discovery using mDNS")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("AetherWave v%s (commit: %s, built: %s)\n", version, commit, date)
		return
	}

	var key []byte
	var err error
	if *keyStr != "" {
		key, err = hex.DecodeString(*keyStr)
		if err != nil || len(key) != 32 {
			fmt.Println("Error: Key must be a 32-byte hex string")
			return
		}
	} else {
		key, err = crypto.GenerateKey()
		if err != nil {
			fmt.Println("Error generating key:", err)
			return
		}
		fmt.Printf("Generated key (share this with peers): %s\n", hex.EncodeToString(key))
	}

	var bootstrapPeers []string
	if *peer != "" {
		bootstrapPeers = []string{*peer}
	}

	bc := blockchain.NewBlockchain()

	// Загружаем блокчейн с диска, если указан chain-file
	if *chainFile != "" {
		if err := bc.LoadFromFile(*chainFile); err != nil {
			fmt.Printf("Warning: failed to load chain from %s: %v\n", *chainFile, err)
		}
	}

	node := network.NewNode(*address, bc, bootstrapPeers)
	node.NetworkKey = key
	node.ChainFile = *chainFile

	if *enableDiscovery {
		fmt.Println("mDNS discovery enabled")
		node.EnableDiscovery()
	}

	if err := node.Start(); err != nil {
		fmt.Printf("Failed to start node: %v\n", err)
		return
	}

	if *peer != "" {
		time.Sleep(1 * time.Second)
		if err := node.ConnectToPeer(*peer); err != nil {
			fmt.Printf("Failed to connect to peer %s: %v\n", *peer, err)
		} else {
			fmt.Printf("Connected to peer %s\n", *peer)
		}
	}

	fmt.Println("Waiting for network stabilization (5 seconds)...")
	time.Sleep(5 * time.Second)

	httpServer := startAPIServer(*apiAddress, bc, node, key, name)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("PANIC in signal handler: %v\n", r)
			}
		}()
		<-sigChan
		fmt.Println("\nShutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		httpServer.Shutdown(ctx)
		node.Stop()
	}()

	fmt.Printf("Welcome, %s! Running on %s (HTTP API on %s)\n", *name, *address, *apiAddress)
	fmt.Println("Commands: send <recipient> <message>, read, peers, blocks, debug, discovery, quit")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			return
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		parts := strings.SplitN(input, " ", 3)

		switch parts[0] {
		case "send":
			handleSendCommand(parts, *name, bc, node, key)
		case "read":
			handleReadCommand(*name, bc, key)
		case "peers":
			handlePeersCommand(node)
		case "blocks":
			handleBlocksCommand(bc)
		case "debug":
			handleDebugCommand(*address, bc, node)
		case "discovery":
			handleDiscoveryCommand(node, parts)
		case "quit":
			fmt.Println("Shutting down...")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			httpServer.Shutdown(ctx)
			node.Stop()
			return
		default:
			fmt.Println("Unknown command. Available: send, read, peers, blocks, debug, discovery, quit")
		}
	}
}

func startAPIServer(addr string, bc *blockchain.Blockchain, node *network.Node, key []byte, name *string) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/message", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Sender    string `json:"sender"`
			Recipient string `json:"recipient"`
			Content   string `json:"content"`
			Key       string `json:"key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}
		if req.Key != "" {
			reqKey, err := hex.DecodeString(req.Key)
			if err != nil || len(reqKey) != 32 {
				http.Error(w, `{"error":"invalid key"}`, http.StatusBadRequest)
				return
			}
			if err := bc.AddMessage(req.Sender, req.Recipient, req.Content, reqKey); err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
				return
			}
		} else {
			if err := bc.AddMessage(req.Sender, req.Recipient, req.Content, key); err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
				return
			}
		}
		lastBlock := bc.GetLastBlock()
		go node.BroadcastBlockWithRetry(lastBlock)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "block": lastBlock})
	})

	mux.HandleFunc("/api/messages", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		username := r.URL.Query().Get("username")
		keyStr := r.URL.Query().Get("key")
		if username == "" {
			http.Error(w, `{"error":"username required"}`, http.StatusBadRequest)
			return
		}
		var msgKey []byte
		if keyStr != "" {
			var err error
			msgKey, err = hex.DecodeString(keyStr)
			if err != nil || len(msgKey) != 32 {
				http.Error(w, `{"error":"invalid key"}`, http.StatusBadRequest)
				return
			}
		} else {
			msgKey = key
		}
		msgs := bc.ReadMessages(username, msgKey)
		var result []map[string]interface{}
		for _, m := range msgs {
			result = append(result, map[string]interface{}{"message": m})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	mux.HandleFunc("/api/blockchain", func(w http.ResponseWriter, r *http.Request) {
		chain := bc.GetChain()
		msgCount := 0
		for _, b := range chain {
			msgCount += len(b.Messages)
		}
		info := map[string]interface{}{
			"blockCount":   len(chain),
			"messageCount": msgCount,
			"difficulty":   bc.Difficulty,
			"lastBlock":    chain[len(chain)-1],
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	})

	mux.HandleFunc("/api/peers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var req struct {
				Address string `json:"address"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
				return
			}
			node.ConnectToPeer(req.Address)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]bool{"success": true})
			return
		}
		peers := node.GetPeers()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(peers)
	})

	mux.HandleFunc("/api/generate-key", func(w http.ResponseWriter, r *http.Request) {
		newKey, err := crypto.GenerateKey()
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"key": hex.EncodeToString(newKey)})
	})

	server := &http.Server{
		Addr:    addr,
		Handler: withCORS(mux),
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in HTTP server: %v\n", r)
			}
		}()
		fmt.Printf("HTTP API server starting on %s\n", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP API server error: %v\n", err)
		}
	}()

	return server
}

func withCORS(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		handler.ServeHTTP(w, r)
	})
}

func handleSendCommand(parts []string, name string, bc *blockchain.Blockchain, node *network.Node, key []byte) {
	fmt.Println("Processing 'send' command...")
	if len(parts) < 3 {
		fmt.Println("Usage: send <recipient> <message>")
		return
	}
	recipient := parts[1]
	message := parts[2]

	startTime := time.Now()
	if err := bc.AddMessage(name, recipient, message, key); err != nil {
		fmt.Printf("Failed to add message: %v\n", err)
		return
	}
	miningDuration := time.Since(startTime)
	fmt.Printf("Message sent and block mined in %v\n", miningDuration)

	lastBlock := bc.GetLastBlock()

	broadcastChan := make(chan bool, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("PANIC in broadcast: %v\n", r)
			}
		}()
		node.BroadcastBlockWithRetry(lastBlock)
		broadcastChan <- true
	}()

	select {
	case <-broadcastChan:
		fmt.Println("Message broadcast completed")
	case <-time.After(5 * time.Second):
		fmt.Println("Timeout broadcasting message - network issue")
	}
}

func handleReadCommand(name string, bc *blockchain.Blockchain, key []byte) {
	fmt.Println("Processing 'read' command...")
	messages := bc.ReadMessages(name, key)

	if len(messages) == 0 {
		fmt.Println("No messages found")
	} else {
		fmt.Println("Your messages:")
		for _, msg := range messages {
			fmt.Println(msg)
		}
	}
	fmt.Println("Read command completed")
}

func handlePeersCommand(node *network.Node) {
	fmt.Println("Processing 'peers' command...")
	peers := node.GetPeers()

	fmt.Println("Known peers:")
	for _, info := range peers {
		status := "inactive"
		if info.Active {
			status = "active"
		}
		fmt.Printf("  %s (Last seen: %v, Status: %s)\n", info.Address, info.LastSeen, status)
	}
	fmt.Println("Peers command completed")
}

func handleBlocksCommand(bc *blockchain.Blockchain) {
	fmt.Println("Processing 'blocks' command...")
	chain := bc.GetChain()

	if len(chain) == 0 {
		fmt.Println("No blocks in the blockchain")
	} else {
		fmt.Println("Blockchain blocks:")
		for _, block := range chain {
			fmt.Printf("Block %d:\n", block.Index)
			fmt.Printf("  Timestamp: %d\n", block.Timestamp)
			fmt.Printf("  Messages: [\n")
			for j, msg := range block.Messages {
				fmt.Printf("    %d: Sender: %s, Recipient: %s, Content: %s, Timestamp: %d\n",
					j, msg.Sender, msg.Recipient, msg.Content, msg.Timestamp)
			}
			fmt.Printf("  ]\n")
			fmt.Printf("  PrevHash: %s\n", block.PrevHash)
			fmt.Printf("  Hash: %s\n", block.Hash)
			fmt.Printf("  Nonce: %d\n", block.Nonce)
			fmt.Println()
		}
	}
	fmt.Println("Blocks command completed")
}

func handleDebugCommand(address string, bc *blockchain.Blockchain, node *network.Node) {
	fmt.Println("Processing 'debug' command...")
	peers := node.GetPeers()
	peerCount := 0
	knownPeerCount := len(peers)

	for _, info := range peers {
		if info.Active {
			peerCount++
		}
	}

	chain := bc.GetChain()
	chainLength := len(chain)
	var lastBlockHash string
	if chainLength > 0 {
		lastBlockHash = chain[chainLength-1].Hash
	} else {
		lastBlockHash = "N/A (empty chain)"
	}

	fmt.Printf("Node %s state:\n", address)
	fmt.Printf("  Chain length: %d\n", chainLength)
	fmt.Printf("  Active peer count: %d\n", peerCount)
	fmt.Printf("  Known peers: %d\n", knownPeerCount)
	fmt.Printf("  Last block hash: %s\n", lastBlockHash)
	fmt.Println("Debug command completed")
}

func handleDiscoveryCommand(node *network.Node, parts []string) {
	fmt.Println("Processing 'discovery' command...")

	if len(parts) > 1 {
		switch parts[1] {
		case "on":
			node.EnableDiscovery()
			fmt.Println("mDNS discovery enabled")
		case "off":
			node.DisableDiscovery()
			fmt.Println("mDNS discovery disabled")
		default:
			fmt.Println("Unknown subcommand. Available: on, off")
		}
	}

	fmt.Println("Discovered nodes:")
	discoveredNodes := node.GetDiscoveredNodes()
	if len(discoveredNodes) == 0 {
		fmt.Println("  No nodes discovered")
	} else {
		for _, addr := range discoveredNodes {
			fmt.Printf("  %s\n", addr)
		}
	}

	fmt.Println("Discovery command completed")
}
