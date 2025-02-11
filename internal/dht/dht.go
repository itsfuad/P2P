package dht

import (
	"bytes"
	"crypto/sha1"
	"sort"
	"sync"
	"time"
)

type Node struct {
	ID       []byte
	Address  string
	LastSeen time.Time
}

type DHT struct {
	Nodes   map[string]*Node
	mu      sync.RWMutex
	LocalID []byte
}

func NewDHT(address string) *DHT {
	id := sha1.Sum([]byte(address))
	return &DHT{
		Nodes:   make(map[string]*Node),
		LocalID: id[:],
	}
}

func (d *DHT) AddNode(node *Node) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Nodes[string(node.ID)] = node
}

func (d *DHT) FindClosestNodes(target []byte, count int) []*Node {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var nodeList []*Node
	for _, node := range d.Nodes {
		nodeList = append(nodeList, node)
	}

	sort.Slice(nodeList, func(i, j int) bool {
		distI := xorDistance(nodeList[i].ID, target)
		distJ := xorDistance(nodeList[j].ID, target)
		return bytes.Compare(distI, distJ) < 0
	})

	if len(nodeList) > count {
		return nodeList[:count]
	}
	return nodeList
}

func xorDistance(a, b []byte) []byte {
	distance := make([]byte, len(a))
	for i := 0; i < len(a); i++ {
		distance[i] = a[i] ^ b[i]
	}
	return distance
}
