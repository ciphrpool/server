package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

func genAESKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("failed to generate random key: %w", err)
	}

	return base64.StdEncoding.EncodeToString(key), nil
}

func DecryptAES(src string, key_b64 string) (string, error) {
	// Decode the base64 string
	encrypted, err := base64.StdEncoding.DecodeString(src)
	if err != nil {
		return "", err
	}

	key, err := base64.StdEncoding.DecodeString(key_b64)
	if err != nil {
		return "", err
	}

	// Extract the nonce and ciphertext
	nonce, ciphertext := encrypted[:12], encrypted[12:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aes_gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %v", err)
	}

	// Decrypt the message
	plaintext, err := aes_gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %v", err)
	}

	return string(plaintext), nil
}

func EncryptAES(src string, key_b64 string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(key_b64)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	aes_gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, 12)
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	ciphertext := aes_gcm.Seal(nil, nonce, []byte(src), nil)
	combined := append(nonce, ciphertext...)
	ciphertext_b64 := base64.StdEncoding.EncodeToString(combined)
	return ciphertext_b64, nil
}

func EncryptAESUrlSafe(src string, key_b64 string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(key_b64)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	aes_gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, 12)
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	ciphertext := aes_gcm.Seal(nil, nonce, []byte(src), nil)
	combined := append(nonce, ciphertext...)
	ciphertext_b64 := base64.URLEncoding.EncodeToString(combined)
	return ciphertext_b64, nil
}
