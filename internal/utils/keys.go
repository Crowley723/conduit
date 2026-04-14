package utils

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

type KeyAlgorithm string

const (
	RSA2048  KeyAlgorithm = "RSA-2048"
	RSA4096  KeyAlgorithm = "RSA-4096"
	ECDSA256 KeyAlgorithm = "ECDSA-P256"
)

func ParseKeyAlgorithm(s string) (KeyAlgorithm, error) {
	switch KeyAlgorithm(s) {
	case RSA2048, RSA4096, ECDSA256:
		return KeyAlgorithm(s), nil
	default:
		return "", errors.New("invalid key algorithm")
	}
}

func GeneratePrivateKey(algorithm KeyAlgorithm) (crypto.PrivateKey, error) {
	switch algorithm {
	case RSA2048, RSA4096:
		privateKey, err := generateRSAPrivateKey(algorithm)
		if err != nil {
			return nil, err
		}

		return privateKey, nil
	case ECDSA256:
		privateKey, err := generateECDSAPrivateKey(algorithm)
		if err != nil {
			return nil, err
		}

		return privateKey, nil
	default:
		return nil, fmt.Errorf("invalid key algorithm: %s", algorithm)
	}
}

func generateRSAPrivateKey(algorithm KeyAlgorithm) (*rsa.PrivateKey, error) {
	var bits int
	if algorithm == RSA2048 {
		bits = 2048
	} else if algorithm == RSA4096 {
		bits = 4096
	} else {
		return nil, errors.New("invalid key algorithm")
	}

	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	return key, nil
}

func generateECDSAPrivateKey(algorithm KeyAlgorithm) (*ecdsa.PrivateKey, error) {
	var curve elliptic.Curve
	if algorithm == ECDSA256 {
		curve = elliptic.P256()
	} else {
		return nil, errors.New("invalid key algorithm")
	}

	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ECDSA key: %w", err)
	}

	return key, nil
}

func PrivateKeyToPEM(privateKey crypto.PrivateKey) ([]byte, error) {
	derBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	pemBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: derBytes,
	}

	return pem.EncodeToMemory(pemBlock), nil
}

func PrivateKeyFromPEM(pemBytes []byte) (crypto.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return key, nil
}
