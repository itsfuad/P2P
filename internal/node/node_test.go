package node_test

import (
	"meshfile/internal/node"
	"testing"
)

const TEST_FILE = "test_file.txt"
const FILE_ADD_ERROR = "Failed to add file: %v"

func TestNewNode(t *testing.T) {
	config := &node.Config{Port: 8080, WebUIPort: 8081}
	n := node.NewNode(config)
	if n == nil {
		t.Fatal("Expected new node to be created")
	}
	if n.GetConfig().Port != 8080 {
		t.Errorf("Expected port 8080, got %d", n.GetConfig().Port)
	}
}

func TestNodeStartStop(t *testing.T) {
	config := &node.Config{Port: 8080, WebUIPort: 8081}
	n := node.NewNode(config)
	err := n.Start()
	if err != nil {
		t.Fatalf("Failed to start node: %v", err)
	}
	n.Stop()
}

func TestNodeAddFile(t *testing.T) {
	config := &node.Config{Port: 8080, WebUIPort: 8081}
	n := node.NewNode(config)
	err := n.AddFile(TEST_FILE)
	if err != nil {
		t.Fatalf(FILE_ADD_ERROR, err)
	}
	files := n.GetFiles()
	if len(files) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(files))
	}
}

func TestNodeEncryptDecryptData(t *testing.T) {
	config := &node.Config{Port: 8080, WebUIPort: 8081}
	n := node.NewNode(config)
	err := n.Start()
	if err != nil {
		t.Fatalf("Failed to start node: %v", err)
	}
	defer n.Stop()

	data := []byte("test data")
	encryptedData, err := n.EncryptData(data)
	if err != nil {
		t.Fatalf("Failed to encrypt data: %v", err)
	}

	decryptedData, err := n.DecryptData(encryptedData)
	if err != nil {
		t.Fatalf("Failed to decrypt data: %v", err)
	}

	if string(decryptedData) != string(data) {
		t.Fatalf("Expected decrypted data to be %s, got %s", data, decryptedData)
	}
}

func TestNodeAddPeer(t *testing.T) {
	config := &node.Config{Port: 8080, WebUIPort: 8081}
	n := node.NewNode(config)
	n.AddPeer("127.0.0.1:8081")
	peers := n.ListPeers()
	if len(peers) != 1 {
		t.Fatalf("Expected 1 peer, got %d", len(peers))
	}
}

func TestNodeRemoveFile(t *testing.T) {
	config := &node.Config{Port: 8080, WebUIPort: 8081}
	n := node.NewNode(config)
	err := n.AddFile(TEST_FILE)
	if err != nil {
		t.Fatalf(FILE_ADD_ERROR, err)
	}
	err = n.RemoveFile(TEST_FILE)
	if err != nil {
		t.Fatalf("Failed to remove file: %v", err)
	}
	files := n.GetFiles()
	if len(files) != 0 {
		t.Fatalf("Expected 0 files, got %d", len(files))
	}
}

func TestNodeGenerateRandomBytes(t *testing.T) {
	config := &node.Config{Port: 8080, WebUIPort: 8081}
	n := node.NewNode(config)
	randomBytes, err := n.GenerateRandomBytes(16)
	if err != nil {
		t.Fatalf("Failed to generate random bytes: %v", err)
	}
	if len(randomBytes) != 16 {
		t.Fatalf("Expected 16 random bytes, got %d", len(randomBytes))
	}
}

func TestNodeGetPeerCount(t *testing.T) {
	config := &node.Config{Port: 8080, WebUIPort: 8081}
	n := node.NewNode(config)
	n.AddPeer("127.0.0.1:8081")
	count := n.GetPeerCount()
	if count != 1 {
		t.Fatalf("Expected 1 peer, got %d", count)
	}
}

func TestNodeGetFileCount(t *testing.T) {
	config := &node.Config{Port: 8080, WebUIPort: 8081}
	n := node.NewNode(config)
	err := n.AddFile(TEST_FILE)
	if err != nil {
		t.Fatalf(FILE_ADD_ERROR, err)
	}
	count := n.GetFileCount()
	if count != 1 {
		t.Fatalf("Expected 1 file, got %d", count)
	}
}
