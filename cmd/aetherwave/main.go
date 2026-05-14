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
	"sync"
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
	peersFile := flag.String("peers-file", "", "Path to persist known peers (JSON); empty means no persistence")
	enableDiscovery := flag.Bool("discovery", false, "Enable automatic node discovery using mDNS")
	showVersion := flag.Bool("version", false, "Show version information")
	// TLS flags
	tlsCert := flag.String("tls-cert", "", "Path to TLS certificate file (enables HTTPS)")
	tlsKey := flag.String("tls-key", "", "Path to TLS key file (required with --tls-cert)")
	// API Auth flags
	apiToken := flag.String("api-token", "", "API authentication token (bearer token for /api/* endpoints)")
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
	node.PeersFile = *peersFile

	// Загружаем пиров с диска, если указан peers-file
	if *peersFile != "" {
		if err := node.LoadPeersFromFile(*peersFile); err != nil {
			fmt.Printf("Warning: failed to load peers from %s: %v\n", *peersFile, err)
		}
	}

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

	httpServer := startAPIServer(*apiAddress, bc, node, key, name, *tlsCert, *tlsKey, *apiToken)

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

func startAPIServer(addr string, bc *blockchain.Blockchain, node *network.Node, key []byte, name *string, tlsCert, tlsKey, apiToken string) *http.Server {
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
		if r.Method == http.MethodGet {
			// DEPRECATED: GET with query params - moved to POST for security
			http.Error(w, `{"error":"use POST /api/messages with JSON body instead of GET with query params"}`, http.StatusMethodNotAllowed)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Username string `json:"username"`
			Key      string `json:"key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}
		if req.Username == "" {
			http.Error(w, `{"error":"username required"}`, http.StatusBadRequest)
			return
		}
		var msgKey []byte
		if req.Key != "" {
			var err error
			msgKey, err = hex.DecodeString(req.Key)
			if err != nil || len(msgKey) != 32 {
				http.Error(w, `{"error":"invalid key"}`, http.StatusBadRequest)
				return
			}
		} else {
			msgKey = key
		}
		msgs := bc.ReadMessages(req.Username, msgKey)
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

	// Health check endpoints
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, что нода запущена
		if node == nil {
			http.Error(w, `{"status":"not ready","error":"node not initialized"}`, http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	})

	// Prometheus metrics endpoint
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		// Базовые метрики
		chain := bc.GetChain()
		msgCount := 0
		for _, b := range chain {
			msgCount += len(b.Messages)
		}
		peers := node.GetPeers()
		activePeers := 0
		for _, p := range peers {
			if p.Active {
				activePeers++
			}
		}

		fmt.Fprintf(w, "# HELP aetherwave_blockchain_blocks_total Total number of blocks in blockchain\n")
		fmt.Fprintf(w, "# TYPE aetherwave_blockchain_blocks_total gauge\n")
		fmt.Fprintf(w, "aetherwave_blockchain_blocks_total %d\n", len(chain))

		fmt.Fprintf(w, "# HELP aetherwave_blockchain_messages_total Total number of messages in blockchain\n")
		fmt.Fprintf(w, "# TYPE aetherwave_blockchain_messages_total gauge\n")
		fmt.Fprintf(w, "aetherwave_blockchain_messages_total %d\n", msgCount)

		fmt.Fprintf(w, "# HELP aetherwave_peers_known_total Total number of known peers\n")
		fmt.Fprintf(w, "# TYPE aetherwave_peers_known_total gauge\n")
		fmt.Fprintf(w, "aetherwave_peers_known_total %d\n", len(peers))

		fmt.Fprintf(w, "# HELP aetherwave_peers_active_total Number of active peers\n")
		fmt.Fprintf(w, "# TYPE aetherwave_peers_active_total gauge\n")
		fmt.Fprintf(w, "aetherwave_peers_active_total %d\n", activePeers)

		fmt.Fprintf(w, "# HELP aetherwave_blockchain_difficulty Current mining difficulty\n")
		fmt.Fprintf(w, "# TYPE aetherwave_blockchain_difficulty gauge\n")
		fmt.Fprintf(w, "aetherwave_blockchain_difficulty %d\n", bc.Difficulty)
	})

	// Применяем middleware: CORS -> Auth -> Rate Limit -> Handler
	handler := withCORS(mux)
	if apiToken != "" {
		handler = withAuth(handler, apiToken)
	}
	handler = withRateLimit(handler)

	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in HTTP server: %v\n", r)
			}
		}()

		if tlsCert != "" && tlsKey != "" {
			fmt.Printf("HTTPS API server starting on %s\n", addr)
			if err := server.ListenAndServeTLS(tlsCert, tlsKey); err != nil && err != http.ErrServerClosed {
				log.Printf("HTTPS API server error: %v\n", err)
			}
		} else {
			fmt.Printf("HTTP API server starting on %s\n", addr)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("HTTP API server error: %v\n", err)
			}
		}
	}()

	return server
}

// withAuth добавляет проверку Bearer токена для API endpoints
func withAuth(handler http.Handler, apiToken string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Health checks не требуют авторизации
		if r.URL.Path == "/health" || r.URL.Path == "/ready" {
			handler.ServeHTTP(w, r)
			return
		}

		// Проверяем Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"authorization required"}`, http.StatusUnauthorized)
			return
		}

		// Ожидаем формат: Bearer <token>
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, `{"error":"invalid authorization format, expected Bearer token"}`, http.StatusUnauthorized)
			return
		}

		if parts[1] != apiToken {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}

		handler.ServeHTTP(w, r)
	})
}

// rateLimiter реализует простой token bucket rate limiter
var rateLimiter = struct {
	mu       sync.Mutex
	tokens   map[string]int
	lastSeen map[string]time.Time
}{
	tokens:   make(map[string]int),
	lastSeen: make(map[string]time.Time),
}

const (
	maxRequestsPerMinute = 60
	burstSize            = 10
)

// withRateLimit добавляет rate limiting по IP
func withRateLimit(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			ip = strings.Split(fwd, ",")[0]
		}

		rateLimiter.mu.Lock()
		now := time.Now()

		// Очищаем старые записи (старше 1 минуты)
		for k, v := range rateLimiter.lastSeen {
			if now.Sub(v) > time.Minute {
				delete(rateLimiter.tokens, k)
				delete(rateLimiter.lastSeen, k)
			}
		}

		// Пополняем токены
		if last, exists := rateLimiter.lastSeen[ip]; exists {
			elapsed := now.Sub(last)
			tokensToAdd := int(elapsed.Seconds()) * maxRequestsPerMinute / 60
			rateLimiter.tokens[ip] = min(rateLimiter.tokens[ip]+tokensToAdd, burstSize)
		} else {
			rateLimiter.tokens[ip] = burstSize
		}

		rateLimiter.lastSeen[ip] = now

		// Проверяем и потребляем токен
		if rateLimiter.tokens[ip] <= 0 {
			rateLimiter.mu.Unlock()
			w.Header().Set("Retry-After", "60")
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}
		rateLimiter.tokens[ip]--
		rateLimiter.mu.Unlock()

		handler.ServeHTTP(w, r)
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func withCORS(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
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
