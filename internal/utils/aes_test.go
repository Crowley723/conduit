package utils

import (
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseEncryptionKey(t *testing.T) {
	var inputValue = "the quick brown fox jumps over the lazy dog"

	keyBytes, err := ParseEncryptionKey(inputValue)

	assert.Nil(t, err)
	assert.NotEmpty(t, keyBytes)
}

func TestParseEncryptionKeyShouldErrorOnEmptyString(t *testing.T) {
	var inputValue = ""

	keyBytes, err := ParseEncryptionKey(inputValue)

	assert.Nil(t, keyBytes)
	assert.Error(t, err, "key cannot be empty")
}

func TestShouldEncryptAndDecryptUsingAES(t *testing.T) {
	var key = sha256.Sum256([]byte("the key"))

	var secret = "abc123"

	encryptedSecret, err := Encrypt([]byte(secret), key[:])
	assert.NoError(t, err, "")

	decryptedSecret, err := Decrypt(encryptedSecret, key[:])

	assert.NoError(t, err, "")
	assert.Equal(t, secret, string(decryptedSecret))
}

func TestDecryptShouldFailWithBadKey(t *testing.T) {
	var key = sha256.Sum256([]byte("the key"))
	var badKey = sha256.Sum256([]byte("not the key"))

	var secret = "the secret"

	encryptedSecret, err := Encrypt([]byte(secret), key[:])
	assert.NoError(t, err, "")

	decryptedSecret, err := Decrypt(encryptedSecret, badKey[:])

	assert.Error(t, err, "unable to decrypt ciphertext: cipher: message authentication failed")
	assert.NotEqual(t, secret, string(decryptedSecret))
}
