package storage

import (
	"context"
	"homelab-dashboard/internal/utils"
	"testing"
)

// TestEncryptionValidationRoundTrip tests the full encryption validation flow
func TestEncryptionValidationRoundTrip(t *testing.T) {
	// Parse a test encryption key
	testKey, err := utils.ParseEncryptionKey("test-encryption-key-for-unit-tests-must-be-long-enough")
	if err != nil {
		t.Fatalf("Failed to parse encryption key: %v", err)
	}

	// Create a mock DatabaseProvider with the test key
	provider := &DatabaseProvider{
		encryptionKey: testKey,
	}

	// Test encrypting and decrypting
	originalData := []byte(EncryptionValidationCheckValue)

	encrypted, err := provider.encrypt(originalData)
	if err != nil {
		t.Fatalf("Failed to encrypt data: %v", err)
	}

	// Verify encrypted data is different from original
	if string(encrypted) == string(originalData) {
		t.Error("Encrypted data should not match original data")
	}

	// Decrypt and verify
	decrypted, err := provider.decrypt(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt data: %v", err)
	}

	if string(decrypted) != string(originalData) {
		t.Errorf("Decrypted data does not match original. Got: %s, Want: %s", string(decrypted), string(originalData))
	}
}

// TestEncryptionWithDifferentKeys tests that data encrypted with one key cannot be decrypted with another
func TestEncryptionWithDifferentKeys(t *testing.T) {
	key1, err := utils.ParseEncryptionKey("first-encryption-key-for-testing-purposes-long-enough")
	if err != nil {
		t.Fatalf("Failed to parse first key: %v", err)
	}

	key2, err := utils.ParseEncryptionKey("second-encryption-key-different-from-first-one-long")
	if err != nil {
		t.Fatalf("Failed to parse second key: %v", err)
	}

	provider1 := &DatabaseProvider{encryptionKey: key1}
	provider2 := &DatabaseProvider{encryptionKey: key2}

	originalData := []byte(EncryptionValidationCheckValue)

	// Encrypt with key1
	encrypted, err := provider1.encrypt(originalData)
	if err != nil {
		t.Fatalf("Failed to encrypt with key1: %v", err)
	}

	// Try to decrypt with key2 - should fail
	_, err = provider2.decrypt(encrypted)
	if err == nil {
		t.Error("Expected decryption with different key to fail, but it succeeded")
	}
}

// TestEncryptDecryptPrivateKey tests encrypting and decrypting a private key
func TestEncryptDecryptPrivateKey(t *testing.T) {
	testKey, err := utils.ParseEncryptionKey("test-key-for-private-key-encryption-testing-long")
	if err != nil {
		t.Fatalf("Failed to parse encryption key: %v", err)
	}

	provider := &DatabaseProvider{encryptionKey: testKey}

	// Generate a test private key
	privateKey, err := utils.GeneratePrivateKey(utils.ECDSA256)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Convert to PEM
	keyPEM, err := utils.PrivateKeyToPEM(privateKey)
	if err != nil {
		t.Fatalf("Failed to convert private key to PEM: %v", err)
	}

	// Encrypt
	encrypted, err := provider.encrypt(keyPEM)
	if err != nil {
		t.Fatalf("Failed to encrypt private key: %v", err)
	}

	// Decrypt
	decrypted, err := provider.decrypt(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt private key: %v", err)
	}

	// Parse back to private key
	parsedKey, err := utils.PrivateKeyFromPEM(decrypted)
	if err != nil {
		t.Fatalf("Failed to parse decrypted private key: %v", err)
	}

	// Verify it's a valid private key (type assertion check)
	if parsedKey == nil {
		t.Error("Decrypted private key is nil")
	}
}

// TestEncryptionValidationCheckValue tests the constant value
func TestEncryptionValidationCheckValue(t *testing.T) {
	expected := "conduit-encryption-validation-v1"
	if EncryptionValidationCheckValue != expected {
		t.Errorf("EncryptionValidationCheckValue mismatch. Got: %s, Want: %s", EncryptionValidationCheckValue, expected)
	}
}

// TestEncryptNilKey tests that encryption fails with nil key
func TestEncryptNilKey(t *testing.T) {
	provider := &DatabaseProvider{encryptionKey: nil}

	_, err := provider.encrypt([]byte("test data"))
	if err == nil {
		t.Error("Expected encryption to fail with nil key, but it succeeded")
	}
}

// TestDecryptNilKey tests that decryption fails with nil key
func TestDecryptNilKey(t *testing.T) {
	provider := &DatabaseProvider{encryptionKey: nil}

	_, err := provider.decrypt([]byte("fake encrypted data"))
	if err == nil {
		t.Error("Expected decryption to fail with nil key, but it succeeded")
	}
}

// TestEncryptEmptyData tests encrypting empty data
func TestEncryptEmptyData(t *testing.T) {
	testKey, err := utils.ParseEncryptionKey("test-key-for-empty-data-encryption-testing-long")
	if err != nil {
		t.Fatalf("Failed to parse encryption key: %v", err)
	}

	provider := &DatabaseProvider{encryptionKey: testKey}

	// Encrypt empty data
	encrypted, err := provider.encrypt([]byte{})
	if err != nil {
		t.Fatalf("Failed to encrypt empty data: %v", err)
	}

	// Decrypt and verify
	decrypted, err := provider.decrypt(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt empty data: %v", err)
	}

	if len(decrypted) != 0 {
		t.Errorf("Expected empty data after decryption, got %d bytes", len(decrypted))
	}
}

// Benchmark encryption performance
func BenchmarkEncrypt(b *testing.B) {
	testKey, _ := utils.ParseEncryptionKey("benchmark-encryption-key-for-performance-testing-long")
	provider := &DatabaseProvider{encryptionKey: testKey}
	data := []byte(EncryptionValidationCheckValue)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = provider.encrypt(data)
	}
}

// Benchmark decryption performance
func BenchmarkDecrypt(b *testing.B) {
	testKey, _ := utils.ParseEncryptionKey("benchmark-decryption-key-for-performance-testing-long")
	provider := &DatabaseProvider{encryptionKey: testKey}
	data := []byte(EncryptionValidationCheckValue)
	encrypted, _ := provider.encrypt(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = provider.decrypt(encrypted)
	}
}

// TestValidateEncryptionKeyMissingKey tests validation with no encryption key configured
func TestValidateEncryptionKeyMissingKey(t *testing.T) {
	provider := &DatabaseProvider{encryptionKey: nil}

	err := provider.ValidateEncryptionKey(context.Background())
	if err == nil {
		t.Error("Expected validation to fail with nil encryption key")
	}
	if err != nil && err.Error() != "encryption key not configured" {
		t.Errorf("Expected 'encryption key not configured' error, got: %v", err)
	}
}
