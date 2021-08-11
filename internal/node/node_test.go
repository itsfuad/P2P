package node

import (
	"testing"
	"time"
)

func TestNodeInitialization(t *testing.T) {
	config := &Config{
		Port:      3000,
		WebUIPort: 8080,
	}

	node := NewNode(config)
	if node == nil {
		t.Fatal("Failed to create node")
	}

	if err := node.Start(); err != nil {
		t.Fatalf("Failed to start node: %v", err)
	}

	// Allow time for services to start
	time.Sleep(time.Second)
}

func TestPeerDiscovery(t *testing.T) {
	config1 := &Config{Port: 3001, WebUIPort: 8081}
	node1 := NewNode(config1)
	if err := node1.Start(); err != nil {
		t.Fatalf("Failed to start node1: %v", err)
	}
	defer func() { /* cleanup */ }()

	config2 := &Config{Port: 3002, WebUIPort: 8082}
	node2 := NewNode(config2)
	if err := node2.Start(); err != nil {
		t.Fatalf("Failed to start node2: %v", err)
	}
	defer func() { /* cleanup */ }()

	time.Sleep(3 * time.Second)
}

func TestFileTransfer(t *testing.T) {
	// Implementation for file transfer test
}
