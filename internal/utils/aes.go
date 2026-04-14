package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

var hkdfSalt = []byte("conduit-encryption-key")
var hkdfInfo = []byte("aes-gcm-key")

func ParseEncryptionKey(input string) ([]byte, error) {
	if input == "" {
		return nil, errors.New("key cannot be empty")
	}

	inputBytes := []byte(input)

	hash := sha256.New

	hkdfReader := hkdf.New(hash, inputBytes, hkdfSalt, hkdfInfo)

	key := make([]byte, 32)
	if _, err := io.ReadFull(hkdfReader, key); err != nil {
		return nil, fmt.Errorf("unable to derive key: %w", err)
	}

	return key, nil
}

func Encrypt(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("unable to create AES cipher: %w", err)
	}

	aesgcm, err := cipher.NewGCMWithRandomNonce(block)
	if err != nil {
		return nil, fmt.Errorf("unable to create GCM: %w", err)
	}

	ciphertext := aesgcm.Seal(nil, nil, plaintext, nil)

	return ciphertext, nil
}

func Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("unable to create AES cipher: %w", err)
	}

	aesgcm, err := cipher.NewGCMWithRandomNonce(block)
	if err != nil {
		return nil, fmt.Errorf("unable to create GCM: %w", err)
	}

	plaintext, err := aesgcm.Open(nil, nil, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to decrypt ciphertext: %w", err)
	}

	return plaintext, nil
}
