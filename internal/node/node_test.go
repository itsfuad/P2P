package node

import (
	"fmt"
	"os"
	"strings"
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
	config1 := &Config{Port: 3003, WebUIPort: 8083}
	node1 := NewNode(config1)
	if err := node1.Start(); err != nil {
		t.Fatalf("Failed to start node1: %v", err)
	}
	defer func() { /* cleanup */ }()

	config2 := &Config{Port: 3004, WebUIPort: 8084}
	node2 := NewNode(config2)
	if err := node2.Start(); err != nil {
		t.Fatalf("Failed to start node2: %v", err)
	}
	defer func() { /* cleanup */ }()

	// Create a test file
	testFileContent := "This is a test file for P2P file transfer."
	testFilePath := "test_file.txt"
	err := os.WriteFile(testFilePath, []byte(testFileContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFilePath)

	// Add the file to node1
	err = node1.AddFile(testFilePath)
	if err != nil {
		t.Fatalf("Failed to add file to node1: %v", err)
	}

	time.Sleep(2 * time.Second)

	// Download the file from node1 to node2
	err = node2.DownloadFile(testFilePath)
	if err != nil && !strings.Contains(err.Error(), "no peers found for file") {
		t.Fatalf("Failed to download file to node2: %v", err)
	}

	// Verify the downloaded file
	downloadedFilePath := "downloaded_test_file.txt"
	downloadedFileContent, err := os.ReadFile(downloadedFilePath)
	if err != nil && !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}
	defer os.Remove(downloadedFilePath)

	if string(downloadedFileContent) != testFileContent {
		t.Fatalf("Downloaded file content does not match original file content")
	}

	// Check if the file was actually downloaded
	if _, err := os.Stat(downloadedFilePath); err == nil {
		fmt.Println("File downloaded successfully")
	} else if os.IsNotExist(err) {
		fmt.Println("File not downloaded")
	} else {
		fmt.Println("Some other error")
	}

	// Clean up
	os.Remove(testFilePath)
	os.Remove(downloadedFilePath)
}
