package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
)

type Encryptor struct {
	publicKey  *rsa.PublicKey
	privateKey *rsa.PrivateKey
}

func NewEncryptor() (*Encryptor, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}
	return &Encryptor{
		publicKey:  &privateKey.PublicKey,
		privateKey: privateKey,
	}, nil
}

func (e *Encryptor) EncryptChunk(data []byte) ([]byte, error) {
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, e.publicKey, data, nil)
	if err != nil {
		return nil, fmt.Errorf("encryption error: %w", err)
	}
	return ciphertext, nil
}

func (e *Encryptor) DecryptChunk(data []byte) ([]byte, error) {
	plaintext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, e.privateKey, data, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption error: %w", err)
	}
	return plaintext, nil
}

func (e *Encryptor) GetPrivateKey() *rsa.PrivateKey {
	return e.privateKey
}

func ComputeHash(data []byte) []byte {
	hasher := sha256.New()
	hasher.Write(data)
	return hasher.Sum(nil)
}
