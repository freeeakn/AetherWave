package network

import (
	"testing"
	"time"

	"github.com/freeeakn/AetherWave/pkg/blockchain"
)

func TestNewMDNSDiscovery(t *testing.T) {
	bc := blockchain.NewBlockchain()
	node := NewNode(":3000", bc, nil)

	discovery := NewMDNSDiscovery(node)

	if discovery == nil {
		t.Fatal("NewMDNSDiscovery returned nil")
	}

	if discovery.node != node {
		t.Error("discovery.node should be set to the provided node")
	}

	if discovery.discoveredMap == nil {
		t.Error("discoveredMap should be initialized")
	}

	if discovery.shutdown == nil {
		t.Error("shutdown channel should be initialized")
	}
}

func TestMDNSDiscoveryGetDiscoveredNodes(t *testing.T) {
	bc := blockchain.NewBlockchain()
	node := NewNode(":3000", bc, nil)
	discovery := NewMDNSDiscovery(node)

	// Изначально список пуст
	nodes := discovery.GetDiscoveredNodes()
	if len(nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(nodes))
	}

	// Добавляем узлы
	discovery.mutex.Lock()
	discovery.discoveredMap["192.168.1.100:3000"] = true
	discovery.discoveredMap["192.168.1.101:3000"] = true
	discovery.mutex.Unlock()

	nodes = discovery.GetDiscoveredNodes()
	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(nodes))
	}

	// Проверяем, что все узлы присутствуют
	found := make(map[string]bool)
	for _, n := range nodes {
		found[n] = true
	}

	if !found["192.168.1.100:3000"] {
		t.Error("expected to find 192.168.1.100:3000")
	}
	if !found["192.168.1.101:3000"] {
		t.Error("expected to find 192.168.1.101:3000")
	}
}

func TestMDNSDiscoveryStop(t *testing.T) {
	bc := blockchain.NewBlockchain()
	node := NewNode(":3000", bc, nil)
	discovery := NewMDNSDiscovery(node)

	// Stop должен быть идемпотентным
	discovery.Stop()

	// Проверяем, что канал закрыт
	select {
	case <-discovery.shutdown:
		// ОК, канал закрыт
	default:
		t.Error("shutdown channel should be closed after Stop()")
	}

	// Повторный вызов Stop не должен паниковать
	discovery.Stop()
}

func TestMDNSDiscoveryConstants(t *testing.T) {
	// Проверяем константы
	if ServiceType != "_aetherwave._tcp" {
		t.Errorf("unexpected ServiceType: %s", ServiceType)
	}

	if ServiceDomain != "local." {
		t.Errorf("unexpected ServiceDomain: %s", ServiceDomain)
	}

	if DiscoveryTimeout != 5*time.Second {
		t.Errorf("unexpected DiscoveryTimeout: %v", DiscoveryTimeout)
	}

	if ServiceTTL != 120 {
		t.Errorf("unexpected ServiceTTL: %d", ServiceTTL)
	}
}
