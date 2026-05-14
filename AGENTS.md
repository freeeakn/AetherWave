# AetherWave — AGENTS.md

## Commands

All through Makefile. Key targets:

| Command | What it does |
|---------|-------------|
| `make build` | Builds `cmd/aetherwave/main.go` → `bin/aetherwave` |
| `make run ARGS="-discovery"` | Run a node with mDNS discovery enabled |
| `make test` | `go test ./...` logs to `test-logs/` |
| `make test-unit` | Tests only `./pkg/...` |
| `make test-integration` | Tests only `./tests/...` |
| `make test-coverage` | All tests + HTML coverage in `coverage/` |
| `make lint` | Runs `go vet` then `golint`/`staticcheck` if installed (skips `scripts/`) |
| `make start-network` | Generates a key via `scripts/generate_key.go`, starts 3 nodes with mDNS |
| `make stop-network` | Kills nodes by PID files in `.pid/` |
| `make dev-env` | Installs `golint`, `staticcheck`, `gocritic` |
| `make profile` | CPU/memory/block profiling via `scripts/profile_performance.go` |
| `make install-deps` | `go mod tidy && go mod download` |

## CLI flags (cmd/aetherwave/main.go)

| Flag | Default | Description |
|------|---------|-------------|
| `--address` | `:3000` | Node P2P address |
| `--api-address` | `:8080` | HTTP API address |
| `--peer` | `""` | Initial peer address to connect to |
| `--name` | `User` | Your username |
| `--key` | `""` | Shared encryption key (hex, 32 bytes); generates one if empty |
| `--chain-file` | `""` | Path to persist blockchain (JSON); no persistence if empty |
| `--discovery` | `false` | Enable mDNS node discovery |
| `--version` | `false` | Show version info and exit |

## Architecture

- **Module**: `github.com/freeeakn/AetherWave` (Go 1.25.0 per `go.mod`, CI uses 1.25)
- **Entrypoint**: `cmd/aetherwave/main.go` — interactive CLI + **HTTP API server** on `:8080` (configurable via `--api-address`)
- **Packages**: `pkg/blockchain/`, `pkg/crypto/`, `pkg/network/`, `pkg/sdk/` (internal SDK via HTTP)
- **External SDK** at `sdk/aetherwave.go` — wraps node HTTP API, used by `mobile/example_app.go`
- **Web UI**: `web/index.html` + `app.js` — talks to node HTTP API at `http://<node>:8080/api/...`
- **mDNS discovery**: `github.com/grandcat/zeroconf`, toggled via `-discovery` flag
- **Block encryption**: AES-256-GCM (authenticated); stored as hex in blockchain and SDKs (hex-encoded 32-byte key, 64 hex chars)
- **P2P encryption**: AES-256-GCM encrypts the entire `NetworkMessage` payload between nodes when `Node.NetworkKey` is set (`--key` flag)
- **Consensus**: PoW — `AddMessage` uses `SimpleMineBlock` (single-threaded, difficulty 4, 60s timeout then keeps mining)
- PoW difficulty is verified in `UpdateChain` and `VerifyChain` (genesis block exempt)
- **Chain persistence**: JSON file via `--chain-file` flag; auto-saves every 10s and on each new block/chain response
- **Peer flood protection**: `MaxPeers` (100) cap in `handlePeerList`; `PeerBroadcastLimit` (30) limits broadcast list size
- **Read timeouts**: `ReadTimeout` (30s) set on P2P connections to prevent hanging
- **HTTP API routes** (all on `--api-address`, default `:8080`):
  - `POST /api/message` — send message (body: `{sender, recipient, content, key?}`)
  - `GET /api/messages?username=&key=` — read messages
  - `GET /api/blockchain` — blockchain info
  - `GET /api/peers`, `POST /api/peers` — peer management
  - `GET /api/generate-key` — generate hex-encoded AES-256 key

## Testing quirks

- `network.DialPeer` and `network.ListenFn` are **exported vars** (`var DialPeer = func...`, `var ListenFn = net.Listen`) for test injection
- Integration tests in `tests/` use `net.Pipe()` based mock infrastructure via `setupTestNodes()`:
  - `pipeListener` accepts in-memory pipe connections
  - `network.DialPeer` creates `net.Pipe()` pairs with distinct labeled addresses and routes the server end to the target listener
  - Tests: `TestNodeCommunication` (2-node), `TestBlockchainSynchronization` (3-node), `TestEndToEndMessaging` (standalone)
- Unit tests in `pkg/blockchain/`, `pkg/crypto/`, `pkg/network/` use `_test.go` in-package tests
- Testing uses `github.com/freeeakn/AetherWave/pkg/blockchain` import (not the SDK)
- Test logs go to `test-logs/<TestName>.log`

## Key gotchas

- **Unified hash**: `CalculateHash` is the single hash function. `SimpleCalculateHash` delegates to it.
- `Blockchain.SaveToFile(path)` and `LoadFromFile(path)` enable chain persistence
- `Node.NetworkKey` ([]byte) enables P2P traffic encryption via `encryptNetworkMessage`/`decryptNetworkMessage`
- `Node.ChainFile` (string) enables automatic chain persistence
- `Node.sendMessage(conn, msg)` is the single send path — handles marshaling + optional encryption + write deadline
- `Node.saveChain()` is called from `broadcastPeerList` ticker (every 10s) and after accepted blocks/chain responses
- Two SDK packages exist: `pkg/sdk/` (internal) and `sdk/` (external); both use `/api/` HTTP paths
- `mobile/` wraps `sdk/` (external SDK)
- Config: `.golangci.yml` includes `goimports`, `misspell`, `revive`, `gofmt`; CI test timeout is 5m (config), script tests use 30s timeout
- Encryption key format: hex-encoded 32 bytes (64 hex chars) across the board
- AES-GCM nonce (12 bytes) is prepended to ciphertext
- `scripts/generate_key.go` and `scripts/profile_performance.go` both use `//go:build ignore` to prevent build conflicts with `go build ./scripts/...`
- P2P connections set `ReadTimeout` (30s) read deadline per decode; timeouts log a warning and continue
