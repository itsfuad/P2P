package node_test

import (
	"io/ioutil"
	"meshfile/internal/node"
	"os"
	"testing"
	"time"
)

const TEST_FILE = "test_file.txt"
const TEST_DATA = "test data"
const FILE_ADD_ERROR = "Failed to add file: %v"

// Test helper function to create a test file
func createTestFile(t *testing.T) {
	err := ioutil.WriteFile(TEST_FILE, []byte(TEST_DATA), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
}

// Test helper function to cleanup test file
func cleanupTestFile(t *testing.T) {
	err := os.Remove(TEST_FILE)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("Failed to cleanup test file: %v", err)
	}
}

// Test helper function to setup a node
func setupNode(t *testing.T) *node.Node {
	config := &node.Config{Port: 0, WebUIPort: 0} // Use dynamic ports
	n := node.NewNode(config)
	err := n.Start()
	if err != nil {
		t.Fatalf("Failed to start node: %v", err)
	}
	return n
}

func TestNewNode(t *testing.T) {
	config := &node.Config{Port: 8080, WebUIPort: 8081}
	n := node.NewNode(config)
	if n == nil {
		t.Fatal("Expected new node to be created")
	}
	if n.GetConfig().Port != 8080 {
		t.Errorf("Expected port 8080, got %d", n.GetConfig().Port)
	}

	defer n.Stop()
}

func TestNodeStartStop(t *testing.T) {
	n := setupNode(t)
	defer n.Stop()
}

func TestNodeAddFile(t *testing.T) {
	n := setupNode(t)
	defer n.Stop()

	createTestFile(t)
	defer cleanupTestFile(t)

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
	n := setupNode(t)
	defer n.Stop()

	// Give the node time to initialize encryption
	time.Sleep(100 * time.Millisecond)

	data := []byte(TEST_DATA)
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
	n := setupNode(t)
	defer n.Stop()

	n.AddPeer("127.0.0.1:8081")
	peers := n.ListPeers()
	if len(peers) != 1 {
		t.Fatalf("Expected 1 peer, got %d", len(peers))
	}
}

func TestNodeRemoveFile(t *testing.T) {
	n := setupNode(t)
	defer n.Stop()

	createTestFile(t)
	defer cleanupTestFile(t)

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
	n := setupNode(t)
	defer n.Stop()

	randomBytes, err := n.GenerateRandomBytes(16)
	if err != nil {
		t.Fatalf("Failed to generate random bytes: %v", err)
	}
	if len(randomBytes) != 16 {
		t.Fatalf("Expected 16 random bytes, got %d", len(randomBytes))
	}
}

func TestNodeGetPeerCount(t *testing.T) {
	n := setupNode(t)
	defer n.Stop()

	n.AddPeer("127.0.0.1:8081")
	count := n.GetPeerCount()
	if count != 1 {
		t.Fatalf("Expected 1 peer, got %d", count)
	}
}

func TestNodeGetFileCount(t *testing.T) {
	n := setupNode(t)
	defer n.Stop()

	createTestFile(t)
	defer cleanupTestFile(t)

	err := n.AddFile(TEST_FILE)
	if err != nil {
		t.Fatalf(FILE_ADD_ERROR, err)
	}

	count := n.GetFileCount()
	if count != 1 {
		t.Fatalf("Expected 1 file, got %d", count)
	}
}
