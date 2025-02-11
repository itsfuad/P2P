package dht

import (
	"testing"
	"time"
)

func TestDHTAddNode(t *testing.T) {
	dht := NewDHT("localhost:3000")
	node := &Node{
		ID:       []byte("testID"),
		Address:  "localhost:3001",
		LastSeen: time.Now(),
	}

	dht.AddNode(node)

	if _, ok := dht.Nodes[string(node.ID)]; !ok {
		t.Errorf("Node not added to DHT")
	}
}

func TestDHTFindClosestNodes(t *testing.T) {
	dht := NewDHT("localhost:3000")

	node1 := &Node{
		ID:       []byte{0x01},
		Address:  "localhost:3001",
		LastSeen: time.Now(),
	}
	node2 := &Node{
		ID:       []byte{0x02},
		Address:  "localhost:3002",
		LastSeen: time.Now(),
	}
	node3 := &Node{
		ID:       []byte{0x03},
		Address:  "localhost:3003",
		LastSeen: time.Now(),
	}

	dht.AddNode(node1)
	dht.AddNode(node2)
	dht.AddNode(node3)

	target := []byte{0x02}
	closestNodes := dht.FindClosestNodes(target, 2)

	if len(closestNodes) != 2 {
		t.Errorf("Expected 2 closest nodes, got %d", len(closestNodes))
	}

	if string(closestNodes[0].ID) != string(node2.ID) {
		t.Errorf("Expected closest node to be node2, got %s", string(closestNodes[0].ID))
	}
}
