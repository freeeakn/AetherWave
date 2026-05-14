# 🌌 AetherWave Blockchain Messenger Network

[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)](https://github.com/freeeakn/AetherWave)
[![Tests](https://img.shields.io/badge/tests-100%25-brightgreen.svg)](https://github.com/freeeakn/AetherWave)
[![Go Version](https://img.shields.io/badge/go-1.23.3-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A secure and decentralized peer-to-peer messaging system built on blockchain technology. Send encrypted messages across a distributed network while maintaining transparency and integrity through blockchain verification.

## ✨ Features

This project demonstrates a blockchain-based messaging system with the following features:

- 📦 Distributed blockchain for message storage
- 🔐 Encrypted messaging using AES-256
- 🌐 Peer-to-peer network with gossip-based peer discovery
- 🔍 Automatic node discovery using mDNS (Multicast DNS)
- ⛏️ Proof-of-work consensus mechanism
- 📊 Performance profiling tools
- 📱 Mobile SDK for integration with mobile applications
- 🖥️ Web interface for easy interaction

The system is split into several main components:

- `blockchain`: Core blockchain implementation with caching
- `crypto`: Encryption/decryption utilities
- `network`: P2P networking and peer discovery
- `sdk`: Software Development Kit for easy integration
- `mobile`: Mobile application integration examples
- `web`: Web interface for interacting with the network
- `cmd/aetherwave`: Application entry point and CLI

## 🚀 Getting Started

### Prerequisites

- Go 1.21 or higher
- Basic understanding of blockchain concepts
- Basic knowledge of Go programming

### Installation

Clone the repository:

```bash
git clone https://github.com/freeeakn/AetherWave.git
cd AetherWave
```

Install dependencies:

```bash
make install-deps
```

Build the project:

```bash
make build
```

## 🏃 Running

### Start a local node

```bash
make run
```

To enable automatic node discovery using mDNS:

```bash
make run ARGS="-discovery"
```

### Start a network of nodes

```bash
make start-network
```

This will start a network of nodes with automatic discovery enabled. The nodes will find each other automatically on the local network.

To stop the network:

```bash
make stop-network
```

### Using Docker

Build and run with Docker:

```bash
make docker-build
make docker-run
```

Clean up Docker resources:

```bash
make docker-clean
```

## 🧪 Testing

AetherWave includes an extensive test suite, including unit and integration tests.

### Run all tests

```bash
make test
```

### Run only unit tests

```bash
make test-unit
```

### Run only integration tests

```bash
make test-integration
```

### Run tests with code coverage

```bash
make test-coverage
```

A code coverage report will be generated in the `coverage/` directory in HTML format.

### Run tests with extended logging

```bash
make test-with-script
```

This option runs tests using a special script that:

- Generates detailed logs for each test
- Creates code coverage reports
- Records error information in a separate file
- Outputs code coverage statistics

All logs and reports are saved in the `test-logs/` and `coverage/` directories.

## 📊 Performance Profiling

AetherWave includes tools for performance profiling:

```bash
make profile
```

This will run profiling for CPU usage, memory usage, and block creation, saving the results in the `profiles/` directory.

## 🔍 Code Quality

Run the linter to check code quality:

```bash
make lint
```

Set up the development environment with necessary tools:

```bash
make dev-env
```

## 📱 Mobile Integration

AetherWave provides an SDK for mobile application integration. Example code is available in the `mobile/` directory.

## 🌐 Web Interface

Access the web interface:

```bash
make web
```

This will open the web interface in your default browser.

## 📂 Project Structure

- `cmd/` - Executable files
    - `aetherwave/` - Main application entry point
- `pkg/` - Core packages
    - `blockchain/` - Blockchain implementation
    - `crypto/` - Cryptographic functions
    - `network/` - Network interaction
        - `node.go`: Node implementation for P2P communication
        - `discovery.go`: Automatic node discovery using mDNS
    - `sdk/` - Software Development Kit (internal)
- `sdk/` - External SDK for third-party integration
- `mobile/` - Mobile integration examples
- `web/` - Web interface
- `tests/` - Integration tests
- `scripts/` - Helper scripts
- `Dockerfile` - Main application Docker image
- `web/Dockerfile` - Web UI Docker image
- `docker-compose.yml` - Multi-node orchestration

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📜 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 📞 Contact

For questions and support, please open an issue on GitHub.

## 🔄 Status

| Component | Status |
|-----------|--------|
| Core Blockchain | ✅ Stable |
| P2P Network | ✅ Stable |
| mDNS Discovery | ✅ Stable |
| Encryption | ✅ Stable |
| Mobile SDK | ✅ Stable |
| Web Interface | 🟡 Beta |
| Documentation | 🟡 In Progress |
