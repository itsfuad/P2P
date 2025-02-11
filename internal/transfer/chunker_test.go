package transfer

import (
	"bytes"
	"strings"
	"testing"
)

func TestFileChunkerSplit(t *testing.T) {
	testString := "This is a test string for file chunking."
	reader := strings.NewReader(testString)
	fileSize := int64(len(testString))

	chunker := NewFileChunker(reader, fileSize)
	err := chunker.Split()
	if err != nil {
		t.Fatalf("Failed to split file into chunks: %v", err)
	}

	if len(chunker.Chunks) == 0 {
		t.Fatalf("No chunks created")
	}

	if int64(len(chunker.Chunks[0].Data)) != fileSize {
		t.Errorf("Chunk size does not match file size, expected %d, got %d", fileSize, len(chunker.Chunks[0].Data))
	}
}

func TestFileChunkerGetChunk(t *testing.T) {
	testString := "This is a test string for file chunking."
	reader := strings.NewReader(testString)
	fileSize := int64(len(testString))

	chunker := NewFileChunker(reader, fileSize)
	err := chunker.Split()
	if err != nil {
		t.Fatalf("Failed to split file into chunks: %v", err)
	}

	chunk, err := chunker.GetChunk(0)
	if err != nil {
		t.Fatalf("Failed to get chunk: %v", err)
	}

	if !bytes.Equal(chunk.Data, []byte(testString)) {
		t.Errorf("Chunk data does not match original string")
	}
}
