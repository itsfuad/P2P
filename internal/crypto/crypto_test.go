package crypto

import (
	"testing"
)

func TestEncryptDecryptChunk(t *testing.T) {
	encryptor, err := NewEncryptor()
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	originalData := []byte("This is a test string for encryption.")

	encryptedData, err := encryptor.EncryptChunk(originalData)
	if err != nil {
		t.Fatalf("Failed to encrypt chunk: %v", err)
	}

	decryptedData, err := encryptor.DecryptChunk(encryptedData)
	if err != nil {
		t.Fatalf("Failed to decrypt chunk: %v", err)
	}

	if string(decryptedData) != string(originalData) {
		t.Errorf("Decrypted data does not match original data")
	}
}

func TestComputeHash(t *testing.T) {
	data := []byte("Test data for hashing")
	hash := ComputeHash(data)

	if len(hash) == 0 {
		t.Errorf("Hash is empty")
	}
}
