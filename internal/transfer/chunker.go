package transfer

import (
	"crypto/sha256"
	"fmt"
	"io"
)

const ChunkSize = 1024 * 1024 // 1MB chunks

type Chunk struct {
	Index uint64
	Hash  []byte
	Data  []byte
}

type FileChunker struct {
	file     io.Reader
	chunks   []*Chunk
	fileSize int64
}

func NewFileChunker(file io.Reader, size int64) *FileChunker {
	return &FileChunker{
		file:     file,
		fileSize: size,
	}
}

func (fc *FileChunker) Split() error {
	var index uint64 = 0
	for {
		chunkData := make([]byte, ChunkSize)
		n, err := fc.file.Read(chunkData)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read chunk: %w", err)
		}
		if n == 0 {
			break
		}

		chunkData = chunkData[:n]
		hash := sha256.Sum256(chunkData)

		chunk := &Chunk{
			Index: index,
			Hash:  hash[:],
			Data:  chunkData,
		}
		fc.chunks = append(fc.chunks, chunk)
		index++

		if err == io.EOF {
			break
		}
	}
	return nil
}

func (fc *FileChunker) GetChunk(index uint64) (*Chunk, error) {
	if index >= uint64(len(fc.chunks)) {
		return nil, fmt.Errorf("chunk index out of range")
	}
	return fc.chunks[index], nil
}
